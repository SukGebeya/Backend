package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func Logger(next http.Handler) http.Handler {
	return middleware.Logger(next)
}

func RequestID(next http.Handler) http.Handler {
	return middleware.RequestID(next)
}

func Recoverer(next http.Handler) http.Handler {
	return middleware.Recoverer(next)
}

// ClientIP resolves the caller's address for logging and rate limiting. With no
// trusted proxies the socket address is used, so X-Forwarded-For cannot be spoofed;
// behind N reverse proxies the Nth-from-last X-Forwarded-For entry is trusted.
func ClientIP(trustedProxies int) func(http.Handler) http.Handler {
	if trustedProxies > 0 {
		return middleware.ClientIPFromXFFTrustedProxies(trustedProxies)
	}
	return middleware.ClientIPFromRemoteAddr
}
