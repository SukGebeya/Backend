package jwt

import (
	"crypto/rsa"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type Manager struct {
	accessPrivateKey  *rsa.PrivateKey
	accessPublicKey   *rsa.PublicKey
	refreshPrivateKey *rsa.PrivateKey
	refreshPublicKey  *rsa.PublicKey
	accessTTL         time.Duration
	refreshTTL        time.Duration
}

func NewManager(accessKeyPair, refreshTokenPair *KeyPair, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		accessPrivateKey:  accessKeyPair.PrivateKey,
		accessPublicKey:   accessKeyPair.PublicKey,
		refreshPrivateKey: refreshTokenPair.PrivateKey,
		refreshPublicKey:  refreshTokenPair.PublicKey,
		accessTTL:         accessTTL,
		refreshTTL:        refreshTTL,
	}
}

func NewManagerFromFiles(accessPrivateKeyPath, accessPublicKeyPath, refreshPrivateKeyPath, refreshPublicKeyPath string, accessTTL, refreshTTL time.Duration) (*Manager, error) {
	accessKeyPair, err := LoadKeyPair(accessPrivateKeyPath, accessPublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load access key pair: %w", err)
	}

	refreshKeyPair, err := LoadKeyPair(refreshPrivateKeyPath, refreshPublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load refresh key pair: %w", err)
	}

	return NewManager(accessKeyPair, refreshKeyPair, accessTTL, refreshTTL), nil
}

func (m *Manager) GenerateAccessToken(userID uuid.UUID, email, role string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
			Issuer:    "go-boilerplate",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.accessPrivateKey)
}

func (m *Manager) GenerateRefreshToken(userID uuid.UUID) (string, string, error) {
	tokenID := uuid.New().String()
	now := time.Now()
	claims := &jwt.RegisteredClaims{
		ID:        tokenID,
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTTL)),
		Issuer:    "go-boilerplate",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(m.refreshPrivateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return signed, tokenID, nil
}

func (m *Manager) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.accessPublicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (m *Manager) ValidateRefreshToken(tokenString string) (uuid.UUID, string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.refreshPublicKey, nil
	})

	if err != nil {
		return uuid.Nil, "", fmt.Errorf("failed to parse refresh token: %w", err)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return uuid.Nil, "", fmt.Errorf("invalid refresh token claims")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("invalid subject in refresh token: %w", err)
	}

	return userID, claims.ID, nil
}

func (m *Manager) GetAccessPublicKey() *rsa.PublicKey {
	return m.accessPublicKey
}

func (m *Manager) GetRefreshPublicKey() *rsa.PublicKey {
	return m.refreshPublicKey
}

func ParseDuration(s string) (time.Duration, error) {
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	if minutes, err := strconv.Atoi(s); err == nil {
		return time.Duration(minutes) * time.Minute, nil
	}

	return 0, fmt.Errorf("invalid duration format: %s", s)
}
