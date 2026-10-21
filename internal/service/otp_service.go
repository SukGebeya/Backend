package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/pkg/phone"
	"github.com/SukGebeya/Backend/pkg/sms"
)

type OTPPurpose string

const (
	OTPPurposeVerifyPhone   OTPPurpose = "verify_phone"
	OTPPurposeResetPassword OTPPurpose = "reset_password"

	otpTTL         = 5 * time.Minute
	otpCooldown    = 60 * time.Second
	otpMaxAttempts = 5
)

type OTPService struct {
	repo    repository.OTPRepository
	sender  sms.Sender
	secret  []byte
	devCode string
}

// NewOTPService creates the SMS one-time-code service. devCode, when set, replaces
// the random code so automated tests can complete verification; config refuses it
// outside development.
func NewOTPService(repo repository.OTPRepository, sender sms.Sender, secret []byte, devCode string) *OTPService {
	return &OTPService{repo: repo, sender: sender, secret: secret, devCode: devCode}
}

func otpKey(purpose OTPPurpose, normalizedPhone string) string {
	return string(purpose) + ":" + normalizedPhone
}

func (s *OTPService) hash(normalizedPhone, code string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(normalizedPhone + ":" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *OTPService) Send(ctx context.Context, rawPhone string, purpose OTPPurpose) error {
	normalized, err := phone.Normalize(rawPhone)
	if err != nil {
		return domainerrors.Validation("phone must be an Ethiopian mobile number")
	}

	key := otpKey(purpose, normalized)
	acquired, err := s.repo.AcquireCooldown(ctx, key, otpCooldown)
	if err != nil {
		return domainerrors.Internal("failed to send code", err)
	}
	if !acquired {
		return domainerrors.RateLimited("please wait a minute before requesting another code")
	}

	code := s.devCode
	if code == "" {
		n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
		if err != nil {
			return domainerrors.Internal("failed to generate code", err)
		}
		code = fmt.Sprintf("%06d", n.Int64())
	}

	if err := s.repo.Store(ctx, key, s.hash(normalized, code), otpTTL); err != nil {
		return domainerrors.Internal("failed to store code", err)
	}

	message := fmt.Sprintf("የገበያ ማረጋገጫ ኮድ፦ %s (ለ5 ደቂቃ ያገለግላል)። Your Gebeya code is %s. It expires in 5 minutes. Never share it.", code, code)
	if err := s.sender.Send(ctx, normalized, message); err != nil {
		_ = s.repo.Delete(ctx, key)
		return domainerrors.Internal("failed to send SMS", err)
	}
	return nil
}

// Verify checks a code and consumes it on success. After otpMaxAttempts wrong
// guesses the code is destroyed and a new one must be requested.
func (s *OTPService) Verify(ctx context.Context, rawPhone string, purpose OTPPurpose, code string) error {
	normalized, err := phone.Normalize(rawPhone)
	if err != nil {
		return domainerrors.Validation("phone must be an Ethiopian mobile number")
	}

	key := otpKey(purpose, normalized)
	stored, err := s.repo.Get(ctx, key)
	if err != nil {
		return domainerrors.Internal("failed to verify code", err)
	}
	if stored == "" {
		return domainerrors.BadRequest("code expired or not requested")
	}

	attempts, err := s.repo.IncrementAttempts(ctx, key, otpTTL)
	if err != nil {
		return domainerrors.Internal("failed to verify code", err)
	}
	if attempts > otpMaxAttempts {
		_ = s.repo.Delete(ctx, key)
		return domainerrors.BadRequest("too many attempts; request a new code")
	}

	if !hmac.Equal([]byte(stored), []byte(s.hash(normalized, code))) {
		return domainerrors.BadRequest("invalid code")
	}

	_ = s.repo.Delete(ctx, key)
	return nil
}
