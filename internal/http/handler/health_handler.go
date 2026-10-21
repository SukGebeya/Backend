package handler

import (
	"net/http"

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/SukGebeya/Backend/pkg/response"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis redis.Cmdable
	blnk  blnk.BlnkClient
}

func NewHealthHandler(db *pgxpool.Pool, redis redis.Cmdable, blnkClient blnk.BlnkClient) *HealthHandler {
	return &HealthHandler{
		db:    db,
		redis: redis,
		blnk:  blnkClient,
	}
}

type HealthResponse struct {
	Status string            `json:"status" example:"healthy"`
	Checks map[string]string `json:"checks"`
}

// Health godoc
// @Summary Health check
// @Description Check the health status of the API and its dependencies. Postgres and Redis are required; a Blnk outage reports "degraded" without failing the check, because browsing and carts still work.
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

	degraded := false
	if h.blnk != nil {
		if err := h.blnk.Ping(r.Context()); err != nil {
			checks["ledger"] = "unhealthy: " + err.Error()
			degraded = true
		} else {
			checks["ledger"] = "ok"
		}
	}

	status := "healthy"
	statusCode := http.StatusOK
	if degraded {
		status = "degraded"
	}
	if !allHealthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	response.JSON(w, statusCode, HealthResponse{
		Status: status,
		Checks: checks,
	})
}
