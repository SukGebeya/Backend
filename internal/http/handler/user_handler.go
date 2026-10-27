package handler

import (
	"net/http"

	"github.com/SukGebeya/Backend/internal/http/middleware"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/pkg/response"
	"github.com/SukGebeya/Backend/pkg/validation"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

type UserResponse struct {
	ID            string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email         *string `json:"email,omitempty" example:"user@example.com"`
	Phone         *string `json:"phone,omitempty" example:"+251911234567"`
	FullName      *string `json:"full_name,omitempty" example:"Abebe Bikila"`
	Role          string  `json:"role" example:"buyer"`
	KYCStatus     string  `json:"kyc_status" example:"pending"`
	PhoneVerified bool    `json:"phone_verified" example:"true"`
	CreatedAt     string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt     string  `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

type UserListResponse struct {
	Users      []UserResponse `json:"users"`
	Total      int64          `json:"total" example:"42"`
	Page       int            `json:"page" example:"1"`
	PerPage    int            `json:"per_page" example:"20"`
	TotalPages int            `json:"total_pages" example:"3"`
}

type UpdateProfileRequest struct {
	Email    *string `json:"email" validate:"omitempty,email" example:"newemail@example.com"`
	Phone    *string `json:"phone" validate:"omitempty,etphone" example:"0911234567"`
	FullName *string `json:"full_name" validate:"omitempty,min=2,max=100" example:"Abebe Bikila"`
}

// GetMe godoc
// @Summary Get current user profile
// @Description Get the authenticated user's profile information
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} UserResponse "User profile"
// @Failure 401 {object} response.ErrorResponse "Authentication required"
// @Failure 404 {object} response.ErrorResponse "User not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /users/me [get]
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, unauthorized("authentication required"))
		return
	}

	user, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toUserResponse(user))
}

// UpdateProfile godoc
// @Summary Update current user profile
// @Description Update the authenticated user's email or phone. At least one of email or phone must be provided.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateProfileRequest true "Profile updates"
// @Success 200 {object} UserResponse "Updated user profile"
// @Failure 400 {object} response.ErrorResponse "Invalid request body or validation error"
// @Failure 401 {object} response.ErrorResponse "Authentication required"
// @Failure 409 {object} response.ErrorResponse "Email or phone already in use"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /users/me [put]
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, unauthorized("authentication required"))
		return
	}

	var req UpdateProfileRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if req.Email == nil && req.Phone == nil && req.FullName == nil {
		response.JSON(w, http.StatusBadRequest, validation.ValidationErrors{
			{Field: "email", Message: "at least one of email, phone or full_name is required"},
		})
		return
	}

	if errs := validation.Validate(req); len(errs) > 0 {
		response.JSON(w, http.StatusBadRequest, errs)
		return
	}

	user, err := h.userService.UpdateProfile(r.Context(), userID, service.UpdateProfileInput{
		Email:    req.Email,
		Phone:    req.Phone,
		FullName: req.FullName,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toUserResponse(user))
}

// ListUsers godoc
// @Summary List all users
// @Description Get a paginated list of all users. Admin access required.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" minimum(1) default(1)
// @Param per_page query int false "Items per page" minimum(1) maximum(100) default(20)
// @Success 200 {object} UserListResponse "List of users"
// @Failure 401 {object} response.ErrorResponse "Authentication required"
// @Failure 403 {object} response.ErrorResponse "Admin access required"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /users [get]
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	opts := parseListOptions(r)

	result, err := h.userService.List(r.Context(), opts)
	if err != nil {
		response.Error(w, err)
		return
	}

	users := make([]UserResponse, len(result.Items))
	for i, u := range result.Items {
		users[i] = *toUserResponse(&u)
	}

	response.JSON(w, http.StatusOK, UserListResponse{
		Users:      users,
		Total:      result.Total,
		Page:       result.Page,
		PerPage:    result.PerPage,
		TotalPages: result.TotalPages,
	})
}

// GetUser godoc
// @Summary Get a user by ID
// @Description Get a specific user's profile. Admin access required.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID" format(uuid)
// @Success 200 {object} UserResponse "User profile"
// @Failure 400 {object} response.ErrorResponse "Invalid user ID format"
// @Failure 401 {object} response.ErrorResponse "Authentication required"
// @Failure 403 {object} response.ErrorResponse "Admin access required"
// @Failure 404 {object} response.ErrorResponse "User not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /users/{id} [get]
func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	user, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toUserResponse(user))
}

// DeleteUser godoc
// @Summary Delete a user
// @Description Permanently delete a user by ID. Admin access required.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID" format(uuid)
// @Success 204 "User deleted successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid user ID format"
// @Failure 401 {object} response.ErrorResponse "Authentication required"
// @Failure 403 {object} response.ErrorResponse "Admin access required"
// @Failure 404 {object} response.ErrorResponse "User not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /users/{id} [delete]
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := parseUUIDParam(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}

	if err := h.userService.Delete(r.Context(), userID); err != nil {
		response.Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
