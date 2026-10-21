package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"

	apperrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/pkg/response"
)

type RateLimiter struct {
	repo   repository.RateLimitRepository
	name   string
	limit  int
	window int64
}

func NewRateLimiter(repo repository.RateLimitRepository, limit int, windowSeconds int64) *RateLimiter {
	return NewNamedRateLimiter(repo, "global", limit, windowSeconds)
}

// NewNamedRateLimiter creates a limiter with its own counters, e.g. a tighter
// limit for login and SMS endpoints alongside the global one.
func NewNamedRateLimiter(repo repository.RateLimitRepository, name string, limit int, windowSeconds int64) *RateLimiter {
	return &RateLimiter{
		repo:   repo,
		name:   name,
		limit:  limit,
		window: windowSeconds,
	}
}

func (rl *RateLimiter) Limit() int {
	return rl.limit
}

func (rl *RateLimiter) Check(ctx context.Context, key string) (count int64, ttl int64, err error) {
	count, err = rl.repo.Increment(ctx, key, rl.window)
	if err != nil {
		return 0, 0, err
	}

	ttl, err = rl.repo.GetTTL(ctx, key)
	if err != nil {
		ttl = rl.window
	}

	return count, ttl, nil
}

func RateLimit(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := limiter.name + ":" + getRateLimitKey(r)

			count, ttl, err := limiter.Check(r.Context(), key)
			if err != nil {
				slog.Warn("rate limit check failed; allowing request", "limiter", limiter.name, "error", err)
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limiter.Limit()))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max(0, limiter.Limit()-int(count))))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(ttl, 10))

			if count > int64(limiter.Limit()) {
				response.Error(w, apperrors.RateLimited("rate limit exceeded"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getRateLimitKey(r *http.Request) string {
	if userID, ok := GetUserID(r.Context()); ok {
		return fmt.Sprintf("user:%s", userID)
	}
	if ip := middleware.GetClientIP(r.Context()); ip != "" {
		return fmt.Sprintf("ip:%s", ip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return fmt.Sprintf("ip:%s", host)
}
