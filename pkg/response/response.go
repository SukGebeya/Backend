package response

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/nate/go-boilerplate/internal/errors"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
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
