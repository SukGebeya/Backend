package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"

	apperrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/pkg/response"
)

const (
	IdempotencyKeyHeader  = "Idempotency-Key"
	idempotencyTTLSeconds = 24 * 60 * 60
	idempotencyLockTTL    = 60
)

type cachedResponse struct {
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

// Idempotency replays the first successful response for a repeated
// Idempotency-Key. Keys are scoped to the caller so one user can never receive
// another user's response, and a lock rejects concurrent duplicates.
func Idempotency(repo repository.IdempotencyRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get(IdempotencyKeyHeader)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()
			scope := "anonymous"
			if userID, ok := GetUserID(ctx); ok {
				scope = userID.String()
			}

			hash := sha256.Sum256([]byte(scope + ":" + r.Method + ":" + r.URL.Path + ":" + key))
			cacheKey := "idempotency:" + hex.EncodeToString(hash[:])
			lockKey := cacheKey + ":lock"

			if replayed := replay(w, r, repo, cacheKey); replayed {
				return
			}

			locked, err := repo.Lock(ctx, lockKey, idempotencyLockTTL)
			if err != nil {
				slog.Error("idempotency lock failed; processing without protection", "error", err)
				next.ServeHTTP(w, r)
				return
			}
			if !locked {
				response.Error(w, apperrors.Conflict("a request with this Idempotency-Key is already in progress"))
				return
			}
			defer func() {
				if err := repo.Unlock(ctx, lockKey); err != nil {
					slog.Error("idempotency unlock failed", "error", err)
				}
			}()

			if replayed := replay(w, r, repo, cacheKey); replayed {
				return
			}

			recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)

			if recorder.status >= 200 && recorder.status < 300 && json.Valid(recorder.body) {
				encoded, err := json.Marshal(cachedResponse{Status: recorder.status, Body: recorder.body})
				if err == nil {
					if err := repo.Set(ctx, cacheKey, string(encoded), idempotencyTTLSeconds); err != nil {
						slog.Error("failed to store idempotent response", "error", err)
					}
				}
			}
		})
	}
}

func replay(w http.ResponseWriter, r *http.Request, repo repository.IdempotencyRepository, cacheKey string) bool {
	stored, err := repo.Get(r.Context(), cacheKey)
	if err != nil || stored == "" {
		return false
	}

	var cached cachedResponse
	if err := json.Unmarshal([]byte(stored), &cached); err != nil || cached.Status == 0 {
		return false
	}

	w.Header().Set("X-Idempotent-Replayed", "true")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(cached.Status)
	_, _ = w.Write(cached.Body)
	return true
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

func IdempotencyRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(IdempotencyKeyHeader) == "" {
			response.Error(w, apperrors.BadRequest("Idempotency-Key header is required"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
