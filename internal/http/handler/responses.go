package handler

import (
	"github.com/nate/go-boilerplate/internal/service"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwidG9rZW5fdHlwZSI6InJlZnJlc2giLCJpYXQiOjE1MTYyMzkwMjJ9..."`
	ExpiresIn    int64  `json:"expires_in" example:"3600"`
}

func toTokenResponse(t *service.AuthTokens) *TokenResponse {
	return &TokenResponse{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpiresIn:    t.ExpiresIn,
	}
}
