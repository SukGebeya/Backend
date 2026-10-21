package handler

import (
	"net/http"

	"github.com/SukGebeya/Backend/internal/domain"
	apperrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/http/middleware"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/pkg/response"
	"github.com/SukGebeya/Backend/pkg/validation"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type RegisterRequest struct {
	Email    *string `json:"email" validate:"omitempty,email" example:"user@example.com"`
	Phone    *string `json:"phone" validate:"omitempty,etphone" example:"0911234567"`
	Password string  `json:"password" validate:"required,min=8" example:"securepassword123"`
	Role     string  `json:"role" validate:"omitempty,oneof=buyer merchant" example:"buyer"`
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
// @Router /register [post]
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

	role := domain.RoleBuyer
	if req.Role != "" {
		role = domain.Role(req.Role)
	}

	result, err := h.authService.Register(r.Context(), service.RegisterInput{
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
		Role:     role,
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
// @Router /login [post]
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
// @Router /refresh [post]
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
// @Router /logout [post]
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
// @Router /logout-all [post]
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

type VerifyPhoneRequest struct {
	Code string `json:"code" validate:"required,len=6,numeric" example:"482913"`
}

type PasswordResetRequest struct {
	Phone string `json:"phone" validate:"required,etphone" example:"0911234567"`
}

type PasswordResetConfirmRequest struct {
	Phone       string `json:"phone" validate:"required,etphone" example:"0911234567"`
	Code        string `json:"code" validate:"required,len=6,numeric" example:"482913"`
	NewPassword string `json:"new_password" validate:"required,min=8" example:"newsecurepassword"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

// SendPhoneCode godoc
// @Summary Send phone verification code
// @Description Texts a 6-digit code (Amharic and English) to the phone on the account. One request per minute.
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 202 {object} MessageResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 429 {object} response.ErrorResponse
// @Router /auth/otp/send [post]
func (h *AuthHandler) SendPhoneCode(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}
	if err := h.authService.SendPhoneVerification(r.Context(), userID); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusAccepted, MessageResponse{Message: "verification code sent"})
}

// VerifyPhone godoc
// @Summary Verify phone number
// @Description Confirms the code sent by SMS and marks the phone number verified. Required before checkout and opening a shop.
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body VerifyPhoneRequest true "Code"
// @Success 200 {object} UserResponse
// @Failure 400 {object} response.ErrorResponse
// @Router /auth/otp/verify [post]
func (h *AuthHandler) VerifyPhone(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}
	var req VerifyPhoneRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}
	user, err := h.authService.VerifyPhone(r.Context(), userID, req.Code)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, toUserResponse(user))
}

// RequestPasswordReset godoc
// @Summary Request a password reset code
// @Description Texts a reset code if the phone belongs to an account. Always answers 202 so it cannot reveal which numbers are registered.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body PasswordResetRequest true "Phone"
// @Success 202 {object} MessageResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 429 {object} response.ErrorResponse
// @Router /auth/password/reset/request [post]
func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req PasswordResetRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}
	if err := h.authService.RequestPasswordReset(r.Context(), req.Phone); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusAccepted, MessageResponse{Message: "if the number is registered, a reset code was sent"})
}

// ConfirmPasswordReset godoc
// @Summary Reset password with SMS code
// @Description Sets a new password and signs the account out everywhere.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body PasswordResetConfirmRequest true "Reset"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} response.ErrorResponse
// @Router /auth/password/reset/confirm [post]
func (h *AuthHandler) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req PasswordResetConfirmRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}
	if err := h.authService.ResetPassword(r.Context(), req.Phone, req.Code, req.NewPassword); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, MessageResponse{Message: "password updated; please log in again"})
}
