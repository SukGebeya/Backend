package http

import (
	"net/http"
	"time"

	_ "github.com/SukGebeya/Backend/docs"
	"github.com/SukGebeya/Backend/internal/http/handler"
	httpmiddleware "github.com/SukGebeya/Backend/internal/http/middleware"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/pkg/jwt"
	"github.com/SukGebeya/Backend/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
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
	jwtManager *jwt.Manager,
	rateLimiter *httpmiddleware.RateLimiter,
	db *pgxpool.Pool,
	redisClient redis.Cmdable,
	appLogger *logger.Logger,
) *Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(httpmiddleware.Timeout(30 * time.Second))
	r.Use(logger.RequestLogger(appLogger))
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(httpmiddleware.RateLimit(rateLimiter))

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	merchantHandler := handler.NewMerchantHandler(merchantService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	productHandler := handler.NewProductHandler(productService, variantService)
	healthHandler := handler.NewHealthHandler(db, redisClient)

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
		})

		// Admin routes
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))
			r.Use(httpmiddleware.Authorize("admin"))

			r.Post("/admin/categories", categoryHandler.CreateCategory)
			r.Put("/admin/categories/{id}", categoryHandler.UpdateCategory)
			r.Delete("/admin/categories/{id}", categoryHandler.DeleteCategory)
		})
	})

	r.Get("/health", healthHandler.Health)

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	return &Router{r}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Mux.ServeHTTP(w, req)
}
