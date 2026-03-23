package jwt

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKeyPair(t *testing.T) {
	tests := []struct {
		name    string
		bits    int
		wantErr bool
	}{
		{
			name:    "generate 2048-bit key pair",
			bits:    2048,
			wantErr: false,
		},
		{
			name:    "generate 4096-bit key pair",
			bits:    4096,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPair, err := GenerateKeyPair(tt.bits)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, keyPair)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, keyPair)
				assert.NotNil(t, keyPair.PrivateKey)
				assert.NotNil(t, keyPair.PublicKey)
				assert.Equal(t, tt.bits, keyPair.PrivateKey.N.BitLen())
			}
		})
	}
}

func TestSaveAndLoadKeyPair(t *testing.T) {
	keyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")

	err = SaveKeyPair(privateKeyPath, publicKeyPath, keyPair)
	require.NoError(t, err)

	assert.FileExists(t, privateKeyPath)
	assert.FileExists(t, publicKeyPath)

	loadedKeyPair, err := LoadKeyPair(privateKeyPath, publicKeyPath)
	require.NoError(t, err)

	assert.NotNil(t, loadedKeyPair.PrivateKey)
	assert.NotNil(t, loadedKeyPair.PublicKey)
	assert.Equal(t, keyPair.PrivateKey.N, loadedKeyPair.PrivateKey.N)
	assert.Equal(t, keyPair.PrivateKey.E, loadedKeyPair.PrivateKey.E)
}

func TestLoadPrivateKey_InvalidPath(t *testing.T) {
	key, err := LoadPrivateKey("nonexistent.pem")
	assert.Error(t, err)
	assert.Nil(t, key)
}

func TestLoadPublicKey_InvalidPath(t *testing.T) {
	key, err := LoadPublicKey("nonexistent.pem")
	assert.Error(t, err)
	assert.Nil(t, key)
}

func TestLoadPrivateKey_InvalidPEM(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "invalid.pem")
	err := os.WriteFile(tmpFile, []byte("not a valid PEM"), 0644)
	require.NoError(t, err)

	key, err := LoadPrivateKey(tmpFile)
	assert.Error(t, err)
	assert.Nil(t, key)
}

func TestLoadPublicKey_InvalidPEM(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "invalid.pem")
	err := os.WriteFile(tmpFile, []byte("not a valid PEM"), 0644)
	require.NoError(t, err)

	key, err := LoadPublicKey(tmpFile)
	assert.Error(t, err)
	assert.Nil(t, key)
}

func TestNewManager(t *testing.T) {
	accessKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	refreshKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	manager := NewManager(accessKeyPair, refreshKeyPair, time.Hour, 24*time.Hour)

	assert.NotNil(t, manager)
	assert.Equal(t, accessKeyPair.PublicKey, manager.GetAccessPublicKey())
	assert.Equal(t, refreshKeyPair.PublicKey, manager.GetRefreshPublicKey())
}

func TestManager_GenerateAndValidateAccessToken(t *testing.T) {
	accessKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	refreshKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	manager := NewManager(accessKeyPair, refreshKeyPair, time.Hour, 24*time.Hour)

	userID := uuid.New()
	email := "test@example.com"
	role := "user"

	token, err := manager.GenerateAccessToken(userID, email, role)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := manager.ValidateAccessToken(token)
	require.NoError(t, err)

	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, "go-boilerplate", claims.Issuer)
	assert.Equal(t, userID.String(), claims.Subject)
}

func TestManager_GenerateAndValidateRefreshToken(t *testing.T) {
	accessKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	refreshKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	manager := NewManager(accessKeyPair, refreshKeyPair, time.Hour, 24*time.Hour)

	userID := uuid.New()

	token, tokenID, err := manager.GenerateRefreshToken(userID)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, tokenID)

	parsedUserID, parsedTokenID, err := manager.ValidateRefreshToken(token)
	require.NoError(t, err)

	assert.Equal(t, userID, parsedUserID)
	assert.Equal(t, tokenID, parsedTokenID)
}

func TestManager_ValidateAccessToken_InvalidToken(t *testing.T) {
	accessKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	refreshKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	manager := NewManager(accessKeyPair, refreshKeyPair, time.Hour, 24*time.Hour)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			token:   "not.a.valid.token",
			wantErr: true,
		},
		{
			name:    "random string",
			token:   uuid.New().String(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ValidateAccessToken(tt.token)
			assert.Error(t, err)
			assert.Nil(t, claims)
		})
	}
}

func TestManager_ValidateRefreshToken_InvalidToken(t *testing.T) {
	accessKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	refreshKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	manager := NewManager(accessKeyPair, refreshKeyPair, time.Hour, 24*time.Hour)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			token:   "not.a.valid.token",
			wantErr: true,
		},
		{
			name:    "random string",
			token:   uuid.New().String(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, tokenID, err := manager.ValidateRefreshToken(tt.token)
			assert.Error(t, err)
			assert.Equal(t, uuid.Nil, userID)
			assert.Empty(t, tokenID)
		})
	}
}

func TestManager_TokenWithWrongKey(t *testing.T) {
	accessKeyPair1, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	refreshKeyPair1, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	manager1 := NewManager(accessKeyPair1, refreshKeyPair1, time.Hour, 24*time.Hour)

	accessKeyPair2, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	refreshKeyPair2, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	manager2 := NewManager(accessKeyPair2, refreshKeyPair2, time.Hour, 24*time.Hour)

	userID := uuid.New()
	token, err := manager1.GenerateAccessToken(userID, "test@example.com", "user")
	require.NoError(t, err)

	claims, err := manager2.ValidateAccessToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestManager_ExpiredToken(t *testing.T) {
	accessKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	refreshKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	manager := NewManager(accessKeyPair, refreshKeyPair, -time.Hour, -time.Hour)

	userID := uuid.New()
	token, err := manager.GenerateAccessToken(userID, "test@example.com", "user")
	require.NoError(t, err)

	claims, err := manager.ValidateAccessToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestManager_AccessTokenUsedAsRefresh(t *testing.T) {
	accessKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	refreshKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	manager := NewManager(accessKeyPair, refreshKeyPair, time.Hour, 24*time.Hour)

	userID := uuid.New()
	accessToken, err := manager.GenerateAccessToken(userID, "test@example.com", "user")
	require.NoError(t, err)

	parsedUserID, tokenID, err := manager.ValidateRefreshToken(accessToken)
	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, parsedUserID)
	assert.Empty(t, tokenID)
}

func TestManager_RefreshTokenUsedAsAccess(t *testing.T) {
	accessKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	refreshKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	manager := NewManager(accessKeyPair, refreshKeyPair, time.Hour, 24*time.Hour)

	userID := uuid.New()
	refreshToken, _, err := manager.GenerateRefreshToken(userID)
	require.NoError(t, err)

	claims, err := manager.ValidateAccessToken(refreshToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "parse hours",
			input:   "2h",
			want:    2 * time.Hour,
			wantErr: false,
		},
		{
			name:    "parse minutes",
			input:   "30m",
			want:    30 * time.Minute,
			wantErr: false,
		},
		{
			name:    "parse seconds",
			input:   "45s",
			want:    45 * time.Second,
			wantErr: false,
		},
		{
			name:    "parse complex duration",
			input:   "1h30m45s",
			want:    1*time.Hour + 30*time.Minute + 45*time.Second,
			wantErr: false,
		},
		{
			name:    "parse number as minutes",
			input:   "60",
			want:    60 * time.Minute,
			wantErr: false,
		},
		{
			name:    "invalid duration",
			input:   "invalid",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDuration(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.want, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetAccessPublicKey(t *testing.T) {
	accessKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	refreshKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	manager := NewManager(accessKeyPair, refreshKeyPair, time.Hour, 24*time.Hour)

	pubKey := manager.GetAccessPublicKey()
	assert.NotNil(t, pubKey)
	assert.Equal(t, accessKeyPair.PublicKey, pubKey)
}

func TestGetRefreshPublicKey(t *testing.T) {
	accessKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	refreshKeyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	manager := NewManager(accessKeyPair, refreshKeyPair, time.Hour, 24*time.Hour)

	pubKey := manager.GetRefreshPublicKey()
	assert.NotNil(t, pubKey)
	assert.Equal(t, refreshKeyPair.PublicKey, pubKey)
}

func TestSavePrivateKey_FilePermissions(t *testing.T) {
	keyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	tmpFile := filepath.Join(t.TempDir(), "private.pem")

	err = SavePrivateKey(tmpFile, keyPair.PrivateKey)
	require.NoError(t, err)

	info, err := os.Stat(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestSavePublicKey_FilePermissions(t *testing.T) {
	keyPair, err := GenerateKeyPair(2048)
	require.NoError(t, err)

	tmpFile := filepath.Join(t.TempDir(), "public.pem")

	err = SavePublicKey(tmpFile, keyPair.PublicKey)
	require.NoError(t, err)

	info, err := os.Stat(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
}
