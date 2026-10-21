package middleware

import (
	"net/http"
	"time"
)

// Timeout bounds request handling time. http.TimeoutHandler buffers the response,
// so a slow handler can never write to the connection after the timeout reply.
func Timeout(timeout time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, `{"error":"Service Unavailable","code":"TIMEOUT","message":"request timed out"}`)
	}
}
