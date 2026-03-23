package handler

import (
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nate/go-boilerplate/pkg/response"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis redis.Cmdable
}

func NewHealthHandler(db *pgxpool.Pool, redis redis.Cmdable) *HealthHandler {
	return &HealthHandler{
		db:    db,
		redis: redis,
	}
}

type HealthResponse struct {
	Status string            `json:"status" example:"healthy"`
	Checks map[string]string `json:"checks"`
}

// Health godoc
// @Summary Health check
// @Description Check the health status of the API and its dependencies
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse "Service is healthy"
// @Failure 503 {object} HealthResponse "Service is unhealthy"
// @Router /health [get]
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]string)
	allHealthy := true

	if err := h.db.Ping(r.Context()); err != nil {
		checks["database"] = "unhealthy: " + err.Error()
		allHealthy = false
	} else {
		checks["database"] = "ok"
	}

	if err := h.redis.Ping(r.Context()).Err(); err != nil {
		checks["redis"] = "unhealthy: " + err.Error()
		allHealthy = false
	} else {
		checks["redis"] = "ok"
	}

	status := "healthy"
	statusCode := http.StatusOK
	if !allHealthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	response.JSON(w, statusCode, HealthResponse{
		Status: status,
		Checks: checks,
	})
}
