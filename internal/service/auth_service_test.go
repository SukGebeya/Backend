package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/SukGebeya/Backend/internal/domain"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/internal/testutil"
)

func TestAuthService_Register_Success(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	redisClient := testutil.NewTestRedis(t)

	userRepo := repository.NewUserRepository(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(redisClient)

	jwtManager := testutil.NewTestJWTManager(t)
	passwordHasher := testutil.NewTestPasswordHasher()

	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtManager, passwordHasher, 7*24*time.Hour)

	email := "newuser@example.com"
	input := service.RegisterInput{
		Email:    &email,
		Password: "Password123!",
		Role:     domain.RoleBuyer,
	}

	result, err := authService.Register(ctx, input)
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if result.User == nil {
		t.Fatal("expected user, got nil")
	}

	if result.User.Email == nil || *result.User.Email != email {
		t.Errorf("expected email %s, got %v", email, result.User.Email)
	}

	if result.User.Role != input.Role {
		t.Errorf("expected role %s, got %s", input.Role, result.User.Role)
	}

	if result.Tokens == nil {
		t.Fatal("expected tokens, got nil")
	}

	if result.Tokens.AccessToken == "" {
		t.Error("expected access token, got empty")
	}

	if result.Tokens.RefreshToken == "" {
		t.Error("expected refresh token, got empty")
	}
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	redisClient := testutil.NewTestRedis(t)

	userRepo := repository.NewUserRepository(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(redisClient)

	jwtManager := testutil.NewTestJWTManager(t)
	passwordHasher := testutil.NewTestPasswordHasher()

	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtManager, passwordHasher, 7*24*time.Hour)

	email := "duplicate@example.com"
	input := service.RegisterInput{
		Email:    &email,
		Password: "Password123!",
		Role:     domain.RoleBuyer,
	}

	_, err := authService.Register(ctx, input)
	if err != nil {
		t.Fatalf("first Register() failed: %v", err)
	}

	_, err = authService.Register(ctx, input)
	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}
}

func TestAuthService_Login_ValidCredentials(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	redisClient := testutil.NewTestRedis(t)

	userRepo := repository.NewUserRepository(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(redisClient)

	jwtManager := testutil.NewTestJWTManager(t)
	passwordHasher := testutil.NewTestPasswordHasher()

	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtManager, passwordHasher, 7*24*time.Hour)

	email := "login@example.com"
	registerInput := service.RegisterInput{
		Email:    &email,
		Password: "Password123!",
		Role:     domain.RoleBuyer,
	}

	_, err := authService.Register(ctx, registerInput)
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	loginInput := service.LoginInput{
		Identifier: email,
		Password:   "Password123!",
	}

	result, err := authService.Login(ctx, loginInput)
	if err != nil {
		t.Fatalf("Login() failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if result.User == nil {
		t.Fatal("expected user, got nil")
	}

	if result.User.Email == nil || *result.User.Email != email {
		t.Errorf("expected email %s, got %v", email, result.User.Email)
	}

	if result.Tokens == nil {
		t.Fatal("expected tokens, got nil")
	}

	if result.Tokens.AccessToken == "" {
		t.Error("expected access token, got empty")
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	redisClient := testutil.NewTestRedis(t)

	userRepo := repository.NewUserRepository(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(redisClient)

	jwtManager := testutil.NewTestJWTManager(t)
	passwordHasher := testutil.NewTestPasswordHasher()

	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtManager, passwordHasher, 7*24*time.Hour)

	email := "wrongpass@example.com"
	registerInput := service.RegisterInput{
		Email:    &email,
		Password: "CorrectPassword123!",
		Role:     domain.RoleBuyer,
	}

	_, err := authService.Register(ctx, registerInput)
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	loginInput := service.LoginInput{
		Identifier: email,
		Password:   "WrongPassword123!",
	}

	_, err = authService.Login(ctx, loginInput)
	if err == nil {
		t.Fatal("expected error for invalid password, got nil")
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	redisClient := testutil.NewTestRedis(t)

	userRepo := repository.NewUserRepository(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(redisClient)

	jwtManager := testutil.NewTestJWTManager(t)
	passwordHasher := testutil.NewTestPasswordHasher()

	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtManager, passwordHasher, 7*24*time.Hour)

	loginInput := service.LoginInput{
		Identifier: "nonexistent@example.com",
		Password:   "Password123!",
	}

	_, err := authService.Login(ctx, loginInput)
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	redisClient := testutil.NewTestRedis(t)

	userRepo := repository.NewUserRepository(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(redisClient)

	jwtManager := testutil.NewTestJWTManager(t)
	passwordHasher := testutil.NewTestPasswordHasher()

	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtManager, passwordHasher, 7*24*time.Hour)

	email := "refresh@example.com"
	registerInput := service.RegisterInput{
		Email:    &email,
		Password: "Password123!",
		Role:     domain.RoleBuyer,
	}

	result, err := authService.Register(ctx, registerInput)
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	tokens, err := authService.Refresh(ctx, result.Tokens.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh() failed: %v", err)
	}

	if tokens == nil {
		t.Fatal("expected tokens, got nil")
	}

	if tokens.AccessToken == "" {
		t.Error("expected access token, got empty")
	}

	if tokens.RefreshToken == "" {
		t.Error("expected refresh token, got empty")
	}
}

func TestAuthService_Logout(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	redisClient := testutil.NewTestRedis(t)

	userRepo := repository.NewUserRepository(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(redisClient)

	jwtManager := testutil.NewTestJWTManager(t)
	passwordHasher := testutil.NewTestPasswordHasher()

	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtManager, passwordHasher, 7*24*time.Hour)

	email := "logout@example.com"
	registerInput := service.RegisterInput{
		Email:    &email,
		Password: "Password123!",
		Role:     domain.RoleBuyer,
	}

	result, err := authService.Register(ctx, registerInput)
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	err = authService.Logout(ctx, result.Tokens.RefreshToken)
	if err != nil {
		t.Fatalf("Logout() failed: %v", err)
	}

	_, err = authService.Refresh(ctx, result.Tokens.RefreshToken)
	if err == nil {
		t.Fatal("expected error when refreshing with logged-out token, got nil")
	}
}
