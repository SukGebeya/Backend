package testutil

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const userIDKey contextKey = "userID"

func ContextWithUserID(userID uuid.UUID) context.Context {
	ctx := context.Background()
	return context.WithValue(ctx, userIDKey, userID)
}

func ContextWithUser() context.Context {
	userID := uuid.New()
	return ContextWithUserID(userID)
}
