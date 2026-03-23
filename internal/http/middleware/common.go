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

func RealIP(next http.Handler) http.Handler {
	return middleware.RealIP(next)
}
