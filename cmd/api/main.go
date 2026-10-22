// Gebeya API
//
// Marketplace backend for small merchants in Addis Ababa: storefronts, carts,
// escrow-backed checkout through Chapa, and Blnk-powered merchant wallets.
//
//	Schemes: http, https
//	Host: localhost:8080
//	BasePath: /api/v1
//	Version: 1.0.0
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	SecurityDefinitions:
//	BearerAuth:
//	  type: apiKey
//	  name: Authorization
//	  in: header
//	  description: Bearer token authentication
//
// swagger:meta
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata" // Africa/Addis_Ababa must resolve even on minimal images

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/SukGebeya/Backend/internal/chapa"
	"github.com/SukGebeya/Backend/internal/config"
	apphttp "github.com/SukGebeya/Backend/internal/http"
	httpmiddleware "github.com/SukGebeya/Backend/internal/http/middleware"
	"github.com/SukGebeya/Backend/internal/jobs"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/pkg/jwt"
	"github.com/SukGebeya/Backend/pkg/logger"
	"github.com/SukGebeya/Backend/pkg/password"
	"github.com/SukGebeya/Backend/pkg/sms"
	"github.com/SukGebeya/Backend/pkg/validation"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

// @title Gebeya API
// @version 1.0
// @description Marketplace backend for small merchants in Addis Ababa: storefronts, carts,
// @description escrow-backed checkout through Chapa, and Blnk-powered merchant wallets.

// @contact.name Gebeya
// @contact.url https://github.com/SukGebeya/Backend

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	if err := run(); err != nil {
		slog.Error("Fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	appLogger := logger.New(logger.Config{
		Format: cfg.Logger.Format,
		Level:  parseLogLevel(cfg.Logger.Level),
	})
	logger.SetDefault(appLogger)

	validation.Init()

	ctx := context.Background()

	pool, err := openDatabase(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := runMigrations(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if err := runRiverMigrations(ctx, pool); err != nil {
		return fmt.Errorf("failed to run River migrations: %w", err)
	}

	redisOpts, err := parseRedisURL(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	redisClient := redis.NewClient(redisOpts)
	defer func() { _ = redisClient.Close() }()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	appLogger.Info("Redis connection established", "addr", redisOpts.Addr)

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	categoryRepo := repository.NewCategoryRepository(pool)
	productRepo := repository.NewProductRepository(pool)
	variantRepo := repository.NewProductVariantRepository(pool)
	imageRepo := repository.NewProductImageRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)
	addressRepo := repository.NewAddressRepository(pool)
	orderRepo := repository.NewOrderRepository(pool)
	paymentRepo := repository.NewPaymentRepository(pool)
	shipmentRepo := repository.NewShipmentRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	payoutRepo := repository.NewPayoutRepository(pool)
	refreshTokenRepo := repository.NewCircuitBreakerRefreshTokenRepository(redisClient)
	rateLimitRepo := repository.NewCircuitBreakerRateLimitRepository(redisClient)
	sessionRepo := repository.NewSessionRepository(redisClient)
	cartRepo := repository.NewCartRepository(pool)
	cartItemRepo := repository.NewCartItemRepository(pool)
	webhookEventRepo := repository.NewWebhookEventRepository(pool)
	txManager := repository.NewTransactionManager(pool)

	appLogger.Info("Circuit breakers enabled for Redis repositories")

	blnkClient, err := blnk.NewClient(blnk.Config{
		BaseURL:          cfg.Blnk.BaseURL,
		SecretKey:        cfg.Blnk.SecretKey,
		ETBLedgerID:      cfg.Blnk.ETBLedgerID,
		EscrowBalanceID:  cfg.Blnk.EscrowBalanceID,
		RevenueBalanceID: cfg.Blnk.RevenueBalanceID,

		ProcessingFeeBalanceID: cfg.Blnk.ProcessingFeeBalanceID,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize Blnk client: %w", err)
	}

	chapaClient := newChapaClient(cfg.Chapa)

	jwtManager, err := jwt.NewManagerFromFiles(
		cfg.JWT.AccessPrivateKeyPath,
		cfg.JWT.AccessPublicKeyPath,
		cfg.JWT.RefreshPrivateKeyPath,
		cfg.JWT.RefreshPublicKeyPath,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize JWT manager: %w", err)
	}

	passwordHasher := password.NewHasher(password.Config{
		Memory:      cfg.Argon2.Memory,
		Iterations:  cfg.Argon2.Iterations,
		Parallelism: cfg.Argon2.Parallelism,
		SaltLength:  cfg.Argon2.SaltLength,
		KeyLength:   cfg.Argon2.KeyLength,
	})

	enqueuer := jobs.NewRiverEnqueuer()
	smsSender := newSMSSender(cfg.SMS)
	otpService := service.NewOTPService(repository.NewOTPRepository(redisClient), smsSender, []byte(cfg.SMS.OTPSecret), cfg.SMS.OTPDevCode)

	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtManager, passwordHasher, cfg.JWT.RefreshTTL).WithOTP(otpService)
	userService := service.NewUserService(userRepo)
	merchantService := service.NewMerchantService(merchantRepo, userRepo, blnkClient, cfg.Platform.RequirePhoneVerification)
	categoryService := service.NewCategoryService(categoryRepo)
	productService := service.NewProductService(productRepo, variantRepo, imageRepo)
	variantService := service.NewProductVariantService(variantRepo, inventoryRepo)
	inventoryService := service.NewInventoryService(inventoryRepo)
	addressService := service.NewAddressService(addressRepo)
	cartService := service.NewCartService(cartRepo, cartItemRepo, sessionRepo, variantRepo, inventoryRepo, enqueuer).WithCatalog(productRepo)
	orderService := service.NewOrderService(orderRepo, merchantRepo, shipmentRepo, walletRepo, paymentRepo, blnkClient, inventoryRepo, txManager, enqueuer).
		WithAutoConfirm(time.Duration(cfg.Platform.AutoConfirmDeliveryDays) * 24 * time.Hour).
		WithAddresses(addressRepo)
	checkoutService := service.NewCheckoutService(cartRepo, cartItemRepo, orderRepo, paymentRepo, addressRepo, inventoryRepo, productRepo, variantRepo, userRepo, merchantRepo, chapaClient, txManager, enqueuer, service.CheckoutConfig{
		FeePercent:               cfg.Platform.FeePercent,
		PaymentTimeout:           cfg.Platform.PaymentTimeout,
		RequirePhoneVerification: cfg.Platform.RequirePhoneVerification,
	})
	paymentService := service.NewPaymentService(service.PaymentDeps{
		Orders:          orderRepo,
		Payments:        paymentRepo,
		Inventory:       inventoryRepo,
		Merchants:       merchantRepo,
		WalletTxns:      walletRepo,
		Blnk:            blnkClient,
		Chapa:           chapaClient,
		TxManager:       txManager,
		Enqueuer:        enqueuer,
		RequireLiveMode: !cfg.IsDevelopment(),
	})
	walletService := service.NewWalletService(merchantRepo, userRepo, walletRepo, payoutRepo, blnkClient, chapaClient, txManager, enqueuer).
		WithPolicy(service.PayoutPolicy{
			MinCents:        cfg.Platform.PayoutMinCents,
			DailyLimitCents: cfg.Platform.PayoutDailyLimitCents,
			Location:        cfg.Platform.Timezone,
			RequireKYC:      true,
		})
	reconciliationService := service.NewReconciliationService(merchantRepo, walletRepo, orderRepo, blnkClient)

	workers := river.NewWorkers()
	jobs.AddWorkers(workers, jobs.WorkerDeps{
		Carts:         cartService,
		Payments:      paymentService,
		Orders:        orderService,
		Payouts:       walletService,
		Reconciler:    reconciliationService,
		SMS:           smsSender,
		WebhookEvents: webhookEventRepo,
		OrderRepo:     orderRepo,
		UserRepo:      userRepo,
	})

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers:      workers,
		ErrorHandler: jobs.NewJobErrorHandler(),
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(jobs.DailyAt{Hour: 3, Location: cfg.Platform.Timezone}, func() (river.JobArgs, *river.InsertOpts) {
				return jobs.ReconcileLedgerArgs{}, nil
			}, nil),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to initialize River client: %w", err)
	}
	enqueuer.Bind(riverClient)

	if err := riverClient.Start(ctx); err != nil {
		return fmt.Errorf("failed to start River client: %w", err)
	}
	appLogger.Info("River job queue started")

	rateLimiter := httpmiddleware.NewRateLimiter(rateLimitRepo, cfg.RateLimit.Requests, int64(cfg.RateLimit.Window.Seconds()))
	authRateLimiter := httpmiddleware.NewNamedRateLimiter(rateLimitRepo, "auth", 10, 60)

	router := apphttp.NewRouter(apphttp.Deps{
		AuthService:      authService,
		UserService:      userService,
		MerchantService:  merchantService,
		CategoryService:  categoryService,
		ProductService:   productService,
		VariantService:   variantService,
		InventoryService: inventoryService,
		AddressService:   addressService,
		CartService:      cartService,
		OrderService:     orderService,
		CheckoutService:  checkoutService,
		PaymentService:   paymentService,
		WalletService:    walletService,
		Reconciliation:   reconciliationService,
		MerchantRepo:     merchantRepo,
		VariantRepo:      variantRepo,
		SessionRepo:      sessionRepo,
		WebhookEvents:    webhookEventRepo,
		TxManager:        txManager,
		JWTManager:       jwtManager,
		RateLimiter:      rateLimiter,
		AuthRateLimiter:  authRateLimiter,
		DB:               pool,
		Redis:            redisClient,
		Logger:           appLogger,
		ChapaClient:      chapaClient,
		BlnkClient:       blnkClient,
		Enqueuer:         enqueuer,
		CORS:             cfg.CORS,
		TrustedProxies:   cfg.Server.TrustedProxyCount,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		appLogger.Info("Starting server", "port", cfg.Port, "environment", cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	var runErr error
	select {
	case <-quit:
	case runErr = <-serverErr:
		appLogger.Error("Server error", "error", runErr)
	}

	appLogger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Error("Server forced to shutdown", "error", err)
	}

	appLogger.Info("Stopping River job queue...")
	if err := riverClient.Stop(ctx); err != nil {
		appLogger.Error("River client shutdown error", "error", err)
	}

	appLogger.Info("Server exited")
	return runErr
}

func openDatabase(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("Database connection pool configured",
		"max_conns", poolConfig.MaxConns,
		"min_conns", poolConfig.MinConns,
		"max_conn_lifetime", poolConfig.MaxConnLifetime,
		"max_conn_idle_time", poolConfig.MaxConnIdleTime,
		"health_check_period", poolConfig.HealthCheckPeriod,
	)
	return pool, nil
}

func newSMSSender(cfg config.SMSConfig) sms.Sender {
	if cfg.Provider == config.SMSProviderAfroMessage {
		return sms.NewAfroMessageSender(sms.AfroMessageConfig{
			BaseURL:      cfg.AfroMessageURL,
			Token:        cfg.AfroMessageToken,
			Sender:       cfg.AfroMessageSender,
			IdentifierID: cfg.AfroMessageFrom,
		})
	}
	slog.Warn("SMS_PROVIDER=log: text messages are written to the log, not sent")
	return sms.LogSender{}
}

func newChapaClient(cfg config.ChapaConfig) chapa.ChapaClient {
	chapaCfg := chapa.Config{
		BaseURL:       cfg.BaseURL,
		SecretKey:     cfg.SecretKey,
		WebhookSecret: cfg.WebhookSecret,
		CallbackURL:   cfg.CallbackURL,
		ReturnURL:     cfg.ReturnURL,
	}
	if cfg.WebhookSecret == "" {
		slog.Warn("CHAPA_WEBHOOK_SECRET is empty: all Chapa webhooks will be rejected")
	}
	if cfg.Mode == config.ChapaModeMock {
		slog.Warn("CHAPA_MODE=mock: payments, transfers and refunds are simulated in memory")
		return chapa.NewFakeClient(chapaCfg)
	}
	return chapa.NewClient(chapaCfg)
}

func runMigrations(databaseURL string) error {
	db, err := goose.OpenDBWithDriver("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open DB: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func runRiverMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("failed to create River migrator: %w", err)
	}

	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			slog.Warn("River table already exists, this may indicate a previous partial migration. Consider resetting River tables if issues persist.")
			return nil
		}
		return fmt.Errorf("failed to run River migrations: %w", err)
	}

	return nil
}

func parseRedisURL(raw string) (*redis.Options, error) {
	if strings.Contains(raw, "://") {
		return redis.ParseURL(raw)
	}
	return &redis.Options{Addr: raw}, nil
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
