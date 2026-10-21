package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	apperrors "github.com/SukGebeya/Backend/internal/errors"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PaginatedResponse struct {
	Items      any   `json:"items"`
	Total      int64 `json:"total" example:"42"`
	Page       int   `json:"page" example:"1"`
	PerPage    int   `json:"per_page" example:"20"`
	TotalPages int   `json:"total_pages" example:"3"`
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func Error(w http.ResponseWriter, err error) {
	if appErr, ok := apperrors.IsAppError(err); ok {
		JSON(w, appErr.HTTPStatus, ErrorResponse{
			Error:   http.StatusText(appErr.HTTPStatus),
			Code:    string(appErr.Code),
			Message: appErr.Message,
		})
		return
	}

	JSON(w, http.StatusInternalServerError, ErrorResponse{
		Error:   http.StatusText(http.StatusInternalServerError),
		Code:    string(apperrors.CodeInternal),
		Message: "An internal error occurred",
	})
}
