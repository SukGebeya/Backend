// Go Boilerplate API
//
// A production-ready Go REST API boilerplate with authentication, notes, and user management.
// Features include JWT authentication, rate limiting, soft deletes, and Redis-based token storage.
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
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/nate/go-boilerplate/internal/config"
	apphttp "github.com/nate/go-boilerplate/internal/http"
	httpmiddleware "github.com/nate/go-boilerplate/internal/http/middleware"
	"github.com/nate/go-boilerplate/internal/repository"
	"github.com/nate/go-boilerplate/internal/service"
	"github.com/nate/go-boilerplate/pkg/jwt"
	"github.com/nate/go-boilerplate/pkg/logger"
	"github.com/nate/go-boilerplate/pkg/password"
	"github.com/nate/go-boilerplate/pkg/validation"
	"github.com/pressly/goose/v3"
)

// @title Go Boilerplate API
// @version 1.0
// @description A production-ready Go REST API boilerplate with authentication, notes, and user management.
// @description Features include JWT authentication, rate limiting, soft deletes, and Redis-based token storage.

// @contact.name API Support
// @contact.url https://github.com/natnael-alemayehu/go-boilerplate
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	appLogger := logger.New(logger.Config{
		Format: cfg.Logger.Format,
		Level:  parseLogLevel(cfg.Logger.Level),
	})
	logger.SetDefault(appLogger)

	validation.Init()

	ctx := context.Background()

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		appLogger.Error("Failed to parse database URL", "error", err)
		os.Exit(1)
	}

	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		appLogger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		appLogger.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}

	appLogger.Info("Database connection pool configured",
		"max_conns", 25,
		"min_conns", 5,
		"max_conn_lifetime", time.Hour,
		"max_conn_idle_time", 30*time.Minute,
		"health_check_period", time.Minute,
	)

	if err := runMigrations(cfg.DatabaseURL); err != nil {
		appLogger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		appLogger.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	appLogger.Info("Redis connection established", "addr", cfg.RedisURL)

	userRepo := repository.NewUserRepository(pool)
	noteRepo := repository.NewNoteRepository(pool)
	refreshTokenRepo := repository.NewCircuitBreakerRefreshTokenRepository(redisClient)
	rateLimitRepo := repository.NewCircuitBreakerRateLimitRepository(redisClient)

	appLogger.Info("Circuit breakers enabled for Redis repositories")

	jwtManager, err := jwt.NewManagerFromFiles(
		cfg.JWT.AccessPrivateKeyPath,
		cfg.JWT.AccessPublicKeyPath,
		cfg.JWT.RefreshPrivateKeyPath,
		cfg.JWT.RefreshPublicKeyPath,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
	)
	if err != nil {
		appLogger.Error("Failed to initialize JWT manager", "error", err)
		os.Exit(1)
	}

	passwordHasher := password.NewHasher(password.Config{
		Memory:      cfg.Argon2.Memory,
		Iterations:  cfg.Argon2.Iterations,
		Parallelism: cfg.Argon2.Parallelism,
		SaltLength:  cfg.Argon2.SaltLength,
		KeyLength:   cfg.Argon2.KeyLength,
	})

	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtManager, passwordHasher, cfg.JWT.RefreshTTL)
	userService := service.NewUserService(userRepo)
	noteService := service.NewNoteService(noteRepo)

	rateLimiter := httpmiddleware.NewRateLimiter(rateLimitRepo, cfg.RateLimit.Requests, int64(cfg.RateLimit.Window.Seconds()))

	router := apphttp.NewRouter(authService, userService, noteService, jwtManager, rateLimiter, pool, redisClient, appLogger)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		appLogger.Info("Starting server", "port", cfg.Port, "environment", cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	appLogger.Info("Server exited")
}

func runMigrations(databaseURL string) error {
	db, err := goose.OpenDBWithDriver("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open DB: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
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
