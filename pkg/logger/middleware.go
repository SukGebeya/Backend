package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) WriteString(s string) (int, error) {
	rw.body.WriteString(s)
	return rw.ResponseWriter.(interface {
		WriteString(string) (int, error)
	}).WriteString(s)
}

func RequestLogger(logger *Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     0,
				body:           bytes.Buffer{},
			}

			var requestBody []byte
			if r.Body != nil {
				requestBody, _ = io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			logFields := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.statusCode,
				"duration_ms", duration.Milliseconds(),
				"remote_addr", r.RemoteAddr,
				"request_id", middleware.GetReqID(r.Context()),
			}

			if r.URL.RawQuery != "" {
				logFields = append(logFields, "query", r.URL.RawQuery)
			}

			if len(requestBody) > 0 && len(requestBody) < 1024 {
				var jsonBody map[string]any
				if err := json.Unmarshal(requestBody, &jsonBody); err == nil {
					if _, ok := jsonBody["password"]; ok {
						jsonBody["password"] = "[REDACTED]"
					}
					logFields = append(logFields, "request_body", jsonBody)
				}
			}

			if rw.statusCode >= 400 && rw.body.Len() > 0 && rw.body.Len() < 1024 {
				var jsonBody map[string]any
				if err := json.Unmarshal(rw.body.Bytes(), &jsonBody); err == nil {
					logFields = append(logFields, "response_body", jsonBody)
				}
			}

			switch {
			case rw.statusCode >= 500:
				logger.Error("Server error", logFields...)
			case rw.statusCode >= 400:
				logger.Warn("Client error", logFields...)
			default:
				logger.Info("Request completed", logFields...)
			}
		})
	}
}
