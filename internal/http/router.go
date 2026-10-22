package http

import (
	"net/http"
	"time"

	_ "github.com/SukGebeya/Backend/docs" // registers the generated swagger spec
	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/SukGebeya/Backend/internal/chapa"
	"github.com/SukGebeya/Backend/internal/config"
	"github.com/SukGebeya/Backend/internal/http/handler"
	httpmiddleware "github.com/SukGebeya/Backend/internal/http/middleware"
	"github.com/SukGebeya/Backend/internal/jobs"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/pkg/jwt"
	"github.com/SukGebeya/Backend/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Router struct {
	*chi.Mux
}

type Deps struct {
	AuthService      *service.AuthService
	UserService      *service.UserService
	MerchantService  *service.MerchantService
	CategoryService  *service.CategoryService
	ProductService   *service.ProductService
	VariantService   *service.ProductVariantService
	InventoryService *service.InventoryService
	AddressService   *service.AddressService
	CartService      *service.CartService
	OrderService     *service.OrderService
	CheckoutService  *service.CheckoutService
	PaymentService   *service.PaymentService
	WalletService    *service.WalletService
	Reconciliation   *service.ReconciliationService

	MerchantRepo  repository.MerchantRepository
	VariantRepo   repository.ProductVariantRepository
	SessionRepo   repository.SessionRepository
	WebhookEvents repository.WebhookEventRepository
	TxManager     repository.TransactionManager

	JWTManager      *jwt.Manager
	RateLimiter     *httpmiddleware.RateLimiter
	AuthRateLimiter *httpmiddleware.RateLimiter
	DB              *pgxpool.Pool
	Redis           redis.Cmdable
	Logger          *logger.Logger
	ChapaClient     chapa.ChapaClient
	BlnkClient      blnk.BlnkClient
	Enqueuer        jobs.Enqueuer
	CORS            config.CORSConfig

	TrustedProxies int
}

func NewRouter(d Deps) *Router {
	r := chi.NewRouter()

	idempotencyRepo := repository.NewIdempotencyRepository(d.Redis)

	r.Use(middleware.RequestID)
	r.Use(httpmiddleware.ClientIP(d.TrustedProxies))
	r.Use(httpmiddleware.Timeout(30 * time.Second))
	r.Use(logger.RequestLogger(d.Logger))
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   d.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Session-ID", "Idempotency-Key"},
		ExposedHeaders:   []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset", "X-Session-ID", "X-Idempotent-Replayed"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(httpmiddleware.RateLimit(d.RateLimiter))

	authHandler := handler.NewAuthHandler(d.AuthService)
	userHandler := handler.NewUserHandler(d.UserService)
	merchantHandler := handler.NewMerchantHandler(d.MerchantService)
	categoryHandler := handler.NewCategoryHandler(d.CategoryService)
	productHandler := handler.NewProductHandler(d.ProductService, d.VariantService, d.InventoryService, d.MerchantService)
	addressHandler := handler.NewAddressHandler(d.AddressService)
	healthHandler := handler.NewHealthHandler(d.DB, d.Redis, d.BlnkClient)
	cartHandler := handler.NewCartHandler(d.CartService, d.VariantRepo)
	orderHandler := handler.NewOrderHandler(d.OrderService, d.PaymentService)
	checkoutHandler := handler.NewCheckoutHandler(d.CheckoutService)
	webhookHandler := handler.NewWebhookHandler(d.ChapaClient, d.WebhookEvents, d.TxManager, d.Enqueuer)
	walletHandler := handler.NewWalletHandler(d.WalletService, d.MerchantService)
	adminHandler := handler.NewAdminHandler(d.WalletService, d.UserService, d.MerchantRepo, d.MerchantService, d.OrderService, d.Reconciliation)
	dashboardHandler := handler.NewDashboardHandler(d.OrderService, d.WalletService)

	auth := httpmiddleware.Auth(d.JWTManager)
	requireMerchant := httpmiddleware.RequireMerchantProfile(d.MerchantService)
	requireAdmin := httpmiddleware.Authorize("admin")
	idempotent := httpmiddleware.Idempotency(idempotencyRepo)
	authLimit := httpmiddleware.RateLimit(d.AuthRateLimiter)

	r.Route("/api/v1", func(r chi.Router) {
		r.With(authLimit).Post("/register", authHandler.Register)
		r.With(authLimit).Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/logout", authHandler.Logout)
		r.With(authLimit).Post("/auth/password/reset/request", authHandler.RequestPasswordReset)
		r.With(authLimit).Post("/auth/password/reset/confirm", authHandler.ConfirmPasswordReset)

		r.Get("/categories", categoryHandler.ListCategories)
		r.Get("/categories/{id}", categoryHandler.GetCategory)
		r.Get("/categories/slug/{slug}", categoryHandler.GetCategoryBySlug)
		r.Get("/products", productHandler.ListProducts)
		r.Get("/products/{slug}", productHandler.GetProduct)
		r.Get("/shops/{shop_slug}", merchantHandler.GetShop)

		r.Post("/webhooks/chapa", webhookHandler.HandleChapa)
		r.Get("/payments/chapa/callback", webhookHandler.ChapaCallback)

		// Carts work for guests (session) and signed-in users (account cart).
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.OptionalAuth(d.JWTManager))
			r.Use(httpmiddleware.Session(d.SessionRepo))
			r.Get("/cart", cartHandler.GetCart)
			r.Post("/cart/items", cartHandler.AddItem)
			r.Put("/cart/items/{variantId}", cartHandler.UpdateItem)
			r.Delete("/cart/items/{variantId}", cartHandler.RemoveItem)
			r.Delete("/cart", cartHandler.ClearCart)
		})

		r.Group(func(r chi.Router) {
			r.Use(auth)

			r.Post("/logout-all", authHandler.LogoutAll)
			r.With(authLimit).Post("/auth/otp/send", authHandler.SendPhoneCode)
			r.With(authLimit).Post("/auth/otp/verify", authHandler.VerifyPhone)
			r.Get("/users/me", userHandler.GetMe)
			r.Put("/users/me", userHandler.UpdateProfile)

			r.Post("/merchant/onboard", merchantHandler.Onboard)

			r.Get("/addresses", addressHandler.ListAddresses)
			r.Post("/addresses", addressHandler.CreateAddress)
			r.Get("/addresses/{id}", addressHandler.GetAddress)
			r.Put("/addresses/{id}", addressHandler.UpdateAddress)
			r.Delete("/addresses/{id}", addressHandler.DeleteAddress)
			r.Put("/addresses/{id}/default", addressHandler.SetDefaultAddress)

			r.With(httpmiddleware.Session(d.SessionRepo)).Post("/cart/merge", cartHandler.MergeCart)

			r.Get("/orders", orderHandler.ListMyOrders)
			r.Get("/orders/{id}", orderHandler.GetOrder)
			r.Post("/orders/{id}/confirm-delivery", orderHandler.ConfirmDelivery)
			r.Post("/orders/{id}/cancel", orderHandler.CancelOrder)
			r.Post("/orders/{id}/payment/refresh", orderHandler.RefreshPayment)
			r.Post("/orders/{id}/dispute", orderHandler.OpenDispute)

			r.With(httpmiddleware.IdempotencyRequired, idempotent).Post("/orders/checkout", checkoutHandler.Checkout)
		})

		r.Group(func(r chi.Router) {
			r.Use(auth)
			r.Use(requireMerchant)

			r.Get("/merchant/dashboard", dashboardHandler.GetDashboard)
			r.Get("/merchant/profile", merchantHandler.GetProfile)
			r.Put("/merchant/profile", merchantHandler.UpdateProfile)

			r.Get("/merchant/products", productHandler.ListMyProducts)
			r.Post("/merchant/products", productHandler.CreateProduct)
			r.Get("/merchant/products/{id}", productHandler.GetMyProduct)
			r.Put("/merchant/products/{id}", productHandler.UpdateProduct)
			r.Delete("/merchant/products/{id}", productHandler.DeleteProduct)
			r.Post("/merchant/products/{id}/variants", productHandler.AddVariant)
			r.Put("/merchant/products/{id}/variants/{variantId}", productHandler.UpdateVariant)
			r.Delete("/merchant/products/{id}/variants/{variantId}", productHandler.DeleteVariant)
			r.Put("/merchant/products/{id}/variants/{variantId}/stock", productHandler.SetVariantStock)
			r.Post("/merchant/products/{id}/images", productHandler.AddProductImage)
			r.Delete("/merchant/products/{id}/images/{imageId}", productHandler.DeleteProductImage)

			r.Get("/merchant/orders", orderHandler.ListMerchantOrders)
			r.Get("/merchant/orders/{id}", orderHandler.GetMerchantOrder)
			r.Put("/merchant/orders/{id}/status", orderHandler.UpdateOrderStatus)
			r.Put("/merchant/orders/{id}/ship", orderHandler.MarkOrderShipped)
			r.Post("/merchant/orders/{id}/cancel", orderHandler.MerchantCancelOrder)

			r.Get("/merchant/wallet", walletHandler.GetWallet)
			r.Get("/merchant/wallet/transactions", walletHandler.ListTransactions)
			r.Get("/merchant/wallet/banks", walletHandler.ListBanks)
			r.Get("/merchant/wallet/payouts", walletHandler.GetMerchantPayouts)
			r.Get("/merchant/wallet/payouts/{payout_id}", walletHandler.GetPayoutByID)
			r.With(httpmiddleware.IdempotencyRequired, idempotent).Post("/merchant/wallet/payout", walletHandler.RequestPayout)
		})

		r.Group(func(r chi.Router) {
			r.Use(auth)
			r.Use(requireAdmin)

			r.Get("/users", userHandler.List)
			r.Get("/users/{id}", userHandler.Get)
			r.Delete("/users/{id}", userHandler.Delete)

			r.Post("/admin/categories", categoryHandler.CreateCategory)
			r.Put("/admin/categories/{id}", categoryHandler.UpdateCategory)
			r.Delete("/admin/categories/{id}", categoryHandler.DeleteCategory)

			r.Get("/admin/users", adminHandler.ListUsers)
			r.Put("/admin/users/{id}/kyc", adminHandler.UpdateKYC)
			r.Get("/admin/merchants", adminHandler.ListMerchants)
			r.Put("/admin/merchants/{id}/verify", adminHandler.VerifyMerchant)
			r.Post("/admin/merchants/{id}/wallet", adminHandler.ProvisionWallet)
			r.Post("/admin/orders/{id}/resolve", adminHandler.ResolveDispute)
			r.Get("/admin/reconciliation", adminHandler.Reconciliation)

			r.Get("/admin/payouts", adminHandler.ListPayouts)
			r.With(idempotent).Post("/admin/payouts/{id}/process", adminHandler.ProcessPayout)
			r.Post("/admin/payouts/{id}/reject", adminHandler.RejectPayout)
		})
	})

	// Kept for webhook URLs registered before the route moved under /api/v1.
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
