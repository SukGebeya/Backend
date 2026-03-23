package middleware

import (
	"context"
	"net/http"
	"time"
)

func Timeout(timeout time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)

			done := make(chan struct{})
			timedOut := false

			go func() {
				defer close(done)
				next.ServeHTTP(w, r)
			}()

			select {
			case <-done:
				return
			case <-ctx.Done():
				timedOut = true
				if ctx.Err() == context.DeadlineExceeded {
					http.Error(w, `{"error":"request timeout"}`, http.StatusRequestTimeout)
					return
				}
			}

			if timedOut {
				http.Error(w, `{"error":"request timeout"}`, http.StatusRequestTimeout)
			}
		})
	}
}
