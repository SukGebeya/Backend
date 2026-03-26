package http

import (
	"net/http"
	"time"

	_ "github.com/SukGebeya/Backend/docs"
	"github.com/SukGebeya/Backend/internal/chapa"
	"github.com/SukGebeya/Backend/internal/http/handler"
	httpmiddleware "github.com/SukGebeya/Backend/internal/http/middleware"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/pkg/jwt"
	"github.com/SukGebeya/Backend/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Router struct {
	*chi.Mux
}

func NewRouter(
	authService *service.AuthService,
	userService *service.UserService,
	merchantService *service.MerchantService,
	categoryService *service.CategoryService,
	productService *service.ProductService,
	variantService *service.ProductVariantService,
	variantRepo repository.ProductVariantRepository,
	addressService *service.AddressService,
	cartService *service.CartService,
	orderService *service.OrderService,
	checkoutService *service.CheckoutService,
	walletService *service.WalletService,
	jwtManager *jwt.Manager,
	rateLimiter *httpmiddleware.RateLimiter,
	db *pgxpool.Pool,
	redisClient redis.Cmdable,
	appLogger *logger.Logger,
	sessionRepo repository.SessionRepository,
	chapaClient *chapa.Client,
	riverClient *river.Client[pgx.Tx],
	adminHandler *handler.AdminHandler,
) *Router {
	r := chi.NewRouter()

	idempotencyRepo := repository.NewIdempotencyRepository(redisClient)

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(httpmiddleware.Timeout(30 * time.Second))
	r.Use(logger.RequestLogger(appLogger))
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Session-ID", "Idempotency-Key"},
		ExposedHeaders:   []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset", "X-Session-ID", "X-Idempotent-Replayed"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(httpmiddleware.RateLimit(rateLimiter))

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	merchantHandler := handler.NewMerchantHandler(merchantService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	productHandler := handler.NewProductHandler(productService, variantService)
	addressHandler := handler.NewAddressHandler(addressService)
	healthHandler := handler.NewHealthHandler(db, redisClient)
	cartHandler := handler.NewCartHandler(cartService, variantRepo)
	orderHandler := handler.NewOrderHandler(orderService)
	checkoutHandler := handler.NewCheckoutHandler(checkoutService)
	webhookHandler := handler.NewWebhookHandler(chapaClient, riverClient)
	walletHandler := handler.NewWalletHandler(walletService, merchantService)

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))
			r.Post("/logout-all", authHandler.LogoutAll)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))

			r.Get("/users/me", userHandler.GetMe)
			r.Put("/users/me", userHandler.UpdateProfile)
		})

		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))
			r.Use(httpmiddleware.Authorize("admin"))

			r.Get("/users", userHandler.List)
			r.Get("/users/{id}", userHandler.Get)
			r.Delete("/users/{id}", userHandler.Delete)
		})

		// Public routes - categories and products
		r.Get("/categories", categoryHandler.ListCategories)
		r.Get("/categories/{id}", categoryHandler.GetCategory)
		r.Get("/categories/slug/{slug}", categoryHandler.GetCategoryBySlug)

		r.Get("/products", productHandler.ListProducts)
		r.Get("/products/{slug}", productHandler.GetProduct)

		r.Get("/shops/{shop_slug}", merchantHandler.GetShop)

		// Merchant onboarding - any authenticated user can onboard
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))
			r.Post("/merchant/onboard", merchantHandler.Onboard)
		})

		// Merchant management routes - requires merchant profile
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))
			r.Use(httpmiddleware.RequireMerchantProfile(merchantService))

			r.Get("/merchant/profile", merchantHandler.GetProfile)
			r.Put("/merchant/profile", merchantHandler.UpdateProfile)

			r.Get("/merchant/products", productHandler.ListMyProducts)
			r.Post("/merchant/products", productHandler.CreateProduct)
			r.Put("/merchant/products/{id}", productHandler.UpdateProduct)
			r.Delete("/merchant/products/{id}", productHandler.DeleteProduct)
			r.Post("/merchant/products/{id}/variants", productHandler.AddVariant)
			r.Put("/merchant/products/{id}/variants/{variantId}", productHandler.UpdateVariant)
			r.Delete("/merchant/products/{id}/variants/{variantId}", productHandler.DeleteVariant)
		})

		// Address routes - authenticated users
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))

			r.Get("/addresses", addressHandler.ListAddresses)
			r.Post("/addresses", addressHandler.CreateAddress)
			r.Get("/addresses/{id}", addressHandler.GetAddress)
			r.Put("/addresses/{id}", addressHandler.UpdateAddress)
			r.Delete("/addresses/{id}", addressHandler.DeleteAddress)
			r.Put("/addresses/{id}/default", addressHandler.SetDefaultAddress)
		})

		// Cart routes (works for both authenticated and guest sessions)
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Session(sessionRepo))
			r.Get("/cart", cartHandler.GetCart)
			r.Post("/cart/items", cartHandler.AddItem)
			r.Put("/cart/items/{variantId}", cartHandler.UpdateItem)
			r.Delete("/cart/items/{variantId}", cartHandler.RemoveItem)
			r.Delete("/cart", cartHandler.ClearCart)
		})

		// Merge guest cart on login (requires authentication)
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))
			r.Use(httpmiddleware.Session(sessionRepo))
			r.Post("/cart/merge", cartHandler.MergeCart)
		})

		// Order routes - authenticated users
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))
			r.Use(httpmiddleware.Session(sessionRepo))

			r.Get("/orders", orderHandler.ListMyOrders)
			r.Get("/orders/{id}", orderHandler.GetOrder)
			r.Post("/orders/{id}/confirm-delivery", orderHandler.ConfirmDelivery)
			r.Post("/orders/{id}/cancel", orderHandler.CancelOrder)
		})

		// Checkout - requires idempotency key for payment safety
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))
			r.Use(httpmiddleware.Session(sessionRepo))
			r.Use(httpmiddleware.IdempotencyRequired)
			r.Use(httpmiddleware.Idempotency(idempotencyRepo))

			r.Post("/orders/checkout", checkoutHandler.Checkout)
		})

		// Merchant order management - requires merchant profile
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))
			r.Use(httpmiddleware.RequireMerchantProfile(merchantService))

			r.Get("/merchant/orders", orderHandler.ListMerchantOrders)
			r.Get("/merchant/orders/{id}", orderHandler.GetMerchantOrder)
			r.Put("/merchant/orders/{id}/status", orderHandler.UpdateOrderStatus)
			r.Put("/merchant/orders/{id}/ship", orderHandler.MarkOrderShipped)
		})

		// Merchant wallet routes - requires merchant profile
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))
			r.Use(httpmiddleware.RequireMerchantProfile(merchantService))

			r.Get("/merchant/wallet", walletHandler.GetWallet)
			r.Get("/merchant/wallet/transactions", walletHandler.ListTransactions)
		})

		// Merchant payout - requires idempotency key
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))
			r.Use(httpmiddleware.RequireMerchantProfile(merchantService))
			r.Use(httpmiddleware.IdempotencyRequired)
			r.Use(httpmiddleware.Idempotency(idempotencyRepo))

			r.Post("/merchant/wallet/payout", walletHandler.RequestPayout)
		})

		// Extended admin routes
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))
			r.Use(httpmiddleware.Authorize("admin"))

			r.Post("/admin/categories", categoryHandler.CreateCategory)
			r.Put("/admin/categories/{id}", categoryHandler.UpdateCategory)
			r.Delete("/admin/categories/{id}", categoryHandler.DeleteCategory)

			r.Get("/admin/payouts", adminHandler.ListPayouts)
			r.Get("/admin/users", adminHandler.ListUsers)
			r.Put("/admin/users/{id}/kyc", adminHandler.UpdateKYC)
			r.Get("/admin/merchants", adminHandler.ListMerchants)
			r.Put("/admin/merchants/{id}/verify", adminHandler.VerifyMerchant)
		})

		// Admin payout processing - idempotency optional but recommended
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))
			r.Use(httpmiddleware.Authorize("admin"))
			r.Use(httpmiddleware.Idempotency(idempotencyRepo))

			r.Post("/admin/payouts/{id}/process", adminHandler.ProcessPayout)
		})
	})

	// Webhook routes - public, no auth required
	r.Post("/webhooks/chapa", webhookHandler.HandleChapa)

	r.Get("/health", healthHandler.Health)

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	return &Router{r}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Mux.ServeHTTP(w, req)
}
