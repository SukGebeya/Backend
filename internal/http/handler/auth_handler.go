package handler

import (
	"net/http"

	"github.com/nate/go-boilerplate/internal/http/middleware"
	"github.com/nate/go-boilerplate/internal/service"
	"github.com/nate/go-boilerplate/pkg/response"
	"github.com/nate/go-boilerplate/pkg/validation"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type RegisterRequest struct {
	Email    *string `json:"email" validate:"omitempty,email" example:"user@example.com"`
	Phone    *string `json:"phone" validate:"omitempty,phone" example:"+1234567890"`
	Password string  `json:"password" validate:"required,min=8" example:"securepassword123"`
}

type LoginRequest struct {
	Identifier string `json:"identifier" validate:"required" example:"user@example.com"`
	Password   string `json:"password" validate:"required" example:"securepassword123"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

type AuthResponse struct {
	User  *UserResponse  `json:"user"`
	Token *TokenResponse `json:"token"`
}

// Register godoc
// @Summary Register a new user
// @Description Create a new user account with email or phone. Returns user info and authentication tokens.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 201 {object} AuthResponse "User registered successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or validation error"
// @Failure 409 {object} response.ErrorResponse "Email or phone already registered"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if req.Email == nil && req.Phone == nil {
		response.JSON(w, http.StatusBadRequest, validation.ValidationErrors{
			{Field: "email", Message: "email or phone is required"},
		})
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	result, err := h.authService.Register(r.Context(), service.RegisterInput{
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, AuthResponse{
		User:  toUserResponse(result.User),
		Token: toTokenResponse(result.Tokens),
	})
}

// Login godoc
// @Summary Login to user account
// @Description Authenticate user with email/phone and password. Returns user info and authentication tokens.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} AuthResponse "Login successful"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or validation error"
// @Failure 401 {object} response.ErrorResponse "Invalid credentials"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	result, err := h.authService.Login(r.Context(), service.LoginInput{
		Identifier: req.Identifier,
		Password:   req.Password,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, AuthResponse{
		User:  toUserResponse(result.User),
		Token: toTokenResponse(result.Tokens),
	})
}

// Refresh godoc
// @Summary Refresh access token
// @Description Get new access and refresh tokens using a valid refresh token.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} TokenResponse "Tokens refreshed successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or validation error"
// @Failure 401 {object} response.ErrorResponse "Invalid or expired refresh token"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	tokens, err := h.authService.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toTokenResponse(tokens))
}

// Logout godoc
// @Summary Logout from current session
// @Description Invalidate the provided refresh token, logging out the user from the current session.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token to invalidate"
// @Success 204 "Logged out successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or validation error"
// @Failure 401 {object} response.ErrorResponse "Invalid refresh token"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	if err := h.authService.Logout(r.Context(), req.RefreshToken); err != nil {
		response.Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// LogoutAll godoc
// @Summary Logout from all sessions
// @Description Invalidate all refresh tokens for the authenticated user, logging out from all devices.
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 204 "Logged out from all sessions successfully"
// @Failure 401 {object} response.ErrorResponse "Authentication required"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/logout-all [post]
func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, unauthorized("authentication required"))
		return
	}

	if err := h.authService.LogoutAll(r.Context(), userID); err != nil {
		response.Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
