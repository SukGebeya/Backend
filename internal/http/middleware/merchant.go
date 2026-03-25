package middleware

import (
	"context"
	"net/http"

	"github.com/SukGebeya/Backend/internal/domain"
	apperrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/pkg/response"
)

const MerchantProfileKey contextKey = "merchantProfile"

func RequireMerchantProfile(merchantService *service.MerchantService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserID(r.Context())
			if !ok {
				response.Error(w, apperrors.Unauthorized("authentication required"))
				return
			}

			profile, err := merchantService.GetByUserID(r.Context(), userID)
			if err != nil {
				response.Error(w, apperrors.Forbidden("merchant profile required - complete onboarding first"))
				return
			}

			ctx := context.WithValue(r.Context(), MerchantProfileKey, profile)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetMerchantProfile(ctx context.Context) (*domain.MerchantProfile, bool) {
	profile, ok := ctx.Value(MerchantProfileKey).(*domain.MerchantProfile)
	return profile, ok
}
