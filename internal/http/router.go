package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/nate/go-boilerplate/docs"
	"github.com/nate/go-boilerplate/internal/http/handler"
	httpmiddleware "github.com/nate/go-boilerplate/internal/http/middleware"
	"github.com/nate/go-boilerplate/internal/service"
	"github.com/nate/go-boilerplate/pkg/jwt"
	"github.com/nate/go-boilerplate/pkg/logger"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Router struct {
	*chi.Mux
}

func NewRouter(
	authService *service.AuthService,
	userService *service.UserService,
	noteService *service.NoteService,
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
	noteHandler := handler.NewNoteHandler(noteService)
	healthHandler := handler.NewHealthHandler(db, redisClient)

	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Group(func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})

		// Authenticated routes - auth
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))
			r.Post("/logout-all", authHandler.LogoutAll)
		})

		// Authenticated routes - user profile
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))

			r.Get("/users/me", userHandler.GetMe)
			r.Put("/users/me", userHandler.UpdateProfile)
		})

		// Admin routes - user management
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))
			r.Use(httpmiddleware.Authorize("admin"))

			r.Get("/users", userHandler.List)
			r.Get("/users/{id}", userHandler.Get)
			r.Delete("/users/{id}", userHandler.Delete)
		})

		// Authenticated routes - notes
		r.Group(func(r chi.Router) {
			r.Use(httpmiddleware.Auth(jwtManager))

			r.Get("/notes", noteHandler.List)
			r.Post("/notes", noteHandler.Create)
			r.Get("/notes/{id}", noteHandler.Get)
			r.Put("/notes/{id}", noteHandler.Update)
			r.Delete("/notes/{id}", noteHandler.Delete)
			r.Post("/notes/{id}/restore", noteHandler.Restore)
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
