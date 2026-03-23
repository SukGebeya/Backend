package testutil

import (
	"context"

	"github.com/google/uuid"
	"github.com/nate/go-boilerplate/internal/http/middleware"
)

func ContextWithUserID(userID uuid.UUID) context.Context {
	ctx := context.Background()
	return context.WithValue(ctx, middleware.UserIDKey, userID)
}

func ContextWithUser() context.Context {
	userID := uuid.New()
	return ContextWithUserID(userID)
}
