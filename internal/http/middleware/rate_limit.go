package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	apperrors "github.com/nate/go-boilerplate/internal/errors"
	"github.com/nate/go-boilerplate/internal/repository"
	"github.com/nate/go-boilerplate/pkg/response"
)

type RateLimiter struct {
	repo   repository.RateLimitRepository
	limit  int
	window int64
}

func NewRateLimiter(repo repository.RateLimitRepository, limit int, windowSeconds int64) *RateLimiter {
	return &RateLimiter{
		repo:   repo,
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
			key := getRateLimitKey(r)

			count, ttl, err := limiter.Check(r.Context(), key)
			if err != nil {
				response.Error(w, apperrors.Internal("rate limit check failed", err))
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
	return fmt.Sprintf("ip:%s", r.RemoteAddr)
}
