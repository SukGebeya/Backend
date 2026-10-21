package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/SukGebeya/Backend/internal/domain"
	apperrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func decodeJSON(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return apperrors.BadRequest("invalid JSON body")
	}
	return nil
}

func parseUUIDParam(r *http.Request, param string) (uuid.UUID, error) {
	idStr := chi.URLParam(r, param)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, apperrors.BadRequest("invalid ID format")
	}
	return id, nil
}

func parseIntQueryParam(r *http.Request, key string, defaultValue int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(val)
	if err != nil || parsed < 1 {
		return defaultValue
	}
	return parsed
}

func parseListOptions(r *http.Request) domain.ListOptions {
	page := parseIntQueryParam(r, "page", 1)
	perPage := parseIntQueryParam(r, "per_page", 20)

	if perPage > 100 {
		perPage = 100
	}

	return domain.ListOptions{
		Page:    page,
		PerPage: perPage,
	}
}

func toUserResponse(u *domain.User) *UserResponse {
	return &UserResponse{
		ID:            u.ID.String(),
		Email:         u.Email,
		Phone:         u.Phone,
		FullName:      u.FullName,
		Role:          string(u.Role),
		KYCStatus:     string(u.KYCStatus),
		PhoneVerified: u.PhoneVerifiedAt != nil,
		CreatedAt:     u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     u.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func unauthorized(msg string) error {
	return apperrors.Unauthorized(msg)
}

func forbidden(msg string) error {
	return apperrors.Forbidden(msg)
}
