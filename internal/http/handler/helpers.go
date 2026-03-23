package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nate/go-boilerplate/internal/domain"
	apperrors "github.com/nate/go-boilerplate/internal/errors"
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

func parseBoolQueryParam(r *http.Request, key string) bool {
	val := r.URL.Query().Get(key)
	return val == "true" || val == "1"
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

func parseNoteListOptions(r *http.Request) domain.NoteListOptions {
	return domain.NoteListOptions{
		ListOptions: parseListOptions(r),
		WithDeleted: parseBoolQueryParam(r, "with_deleted"),
	}
}

func badRequest(msg string) error {
	return apperrors.BadRequest(msg)
}

func unauthorized(msg string) error {
	return apperrors.Unauthorized(msg)
}

func validationError(msg string) error {
	return apperrors.Validation(msg)
}

func toUserResponse(u *domain.User) *UserResponse {
	return &UserResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		Phone:     u.Phone,
		Role:      string(u.Role),
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: u.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toNoteResponse(n *domain.Note) *NoteResponse {
	var deletedAt *string
	if n.DeletedAt != nil {
		formatted := n.DeletedAt.Format("2006-01-02T15:04:05Z07:00")
		deletedAt = &formatted
	}
	return &NoteResponse{
		ID:        n.ID.String(),
		UserID:    n.UserID.String(),
		Title:     n.Title,
		Content:   n.Content,
		CreatedAt: n.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: n.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		DeletedAt: deletedAt,
	}
}
