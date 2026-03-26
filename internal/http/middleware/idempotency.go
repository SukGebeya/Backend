package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	apperrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/pkg/response"
)

type IdempotencyKey string

const IdempotencyKeyHeader = "Idempotency-Key"

func Idempotency(redisRepo repository.IdempotencyRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get(IdempotencyKeyHeader)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()

			method := r.Method
			path := r.URL.Path

			hash := sha256.Sum256([]byte(method + ":" + path + ":" + key))
			cacheKey := "idempotency:" + hex.EncodeToString(hash[:])

			result, err := redisRepo.Get(ctx, cacheKey)
			if err == nil && result != "" {
				w.Header().Set("X-Idempotent-Replayed", "true")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(result))
				return
			}

			responseRecorder := &responseRecorder{
				ResponseWriter: w,
				body:           make([]byte, 0),
			}

			ctx = context.WithValue(ctx, IdempotencyKey("cache_key"), cacheKey)
			ctx = context.WithValue(ctx, IdempotencyKey("redis_repo"), redisRepo)

			next.ServeHTTP(responseRecorder, r.WithContext(ctx))

			if responseRecorder.status >= 200 && responseRecorder.status < 300 {
				ttl := int64(24 * time.Hour)
				_ = redisRepo.Set(ctx, cacheKey, string(responseRecorder.body), ttl)
			}
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	body   []byte
	status int
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func GetIdempotencyCacheKey(ctx context.Context) (string, bool) {
	key, ok := ctx.Value(IdempotencyKey("cache_key")).(string)
	return key, ok
}

func IdempotencyRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get(IdempotencyKeyHeader)
		if key == "" {
			response.Error(w, apperrors.BadRequest("Idempotency-Key header is required"))
			return
		}

		next.ServeHTTP(w, r)
	})
}
