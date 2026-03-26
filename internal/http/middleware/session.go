package middleware

import (
	"context"
	"net/http"

	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/google/uuid"
)

const SessionIDKey contextKey = "session_id"

func Session(sessionRepo repository.SessionRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionID := r.Header.Get("X-Session-ID")

			if sessionID == "" {
				cookie, err := r.Cookie("session_id")
				if err == nil {
					sessionID = cookie.Value
				}
			}

			if sessionID != "" {
				valid, err := sessionRepo.Get(r.Context(), sessionID)
				if err != nil || !valid {
					sessionID = uuid.New().String()
					ttl := int64(7 * 24 * 60 * 60)
					if err := sessionRepo.Create(r.Context(), sessionID, ttl); err != nil {
						http.Error(w, "failed to create session", http.StatusInternalServerError)
						return
					}
				}
			} else {
				sessionID = uuid.New().String()
				ttl := int64(7 * 24 * 60 * 60)
				if err := sessionRepo.Create(r.Context(), sessionID, ttl); err != nil {
					http.Error(w, "failed to create session", http.StatusInternalServerError)
					return
				}
			}

			ctx := context.WithValue(r.Context(), SessionIDKey, sessionID)
			w.Header().Set("X-Session-ID", sessionID)

			http.SetCookie(w, &http.Cookie{
				Name:     "session_id",
				Value:    sessionID,
				Path:     "/",
				MaxAge:   7 * 24 * 60 * 60,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetSessionID(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(SessionIDKey).(string)
	return sessionID, ok
}
