package password

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasher_Hash(t *testing.T) {
	tests := []struct {
		name     string
		password string
		config   Config
		wantErr  bool
	}{
		{
			name:     "hash password with default config",
			password: "mysecretpassword",
			config:   DefaultConfig(),
			wantErr:  false,
		},
		{
			name:     "hash empty password",
			password: "",
			config:   DefaultConfig(),
			wantErr:  false,
		},
		{
			name:     "hash long password",
			password: strings.Repeat("a", 1000),
			config:   DefaultConfig(),
			wantErr:  false,
		},
		{
			name:     "hash with custom config",
			password: "test123",
			config: Config{
				Memory:      32768,
				Iterations:  2,
				Parallelism: 1,
				SaltLength:  16,
				KeyLength:   32,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHasher(tt.config)
			hash, err := h.Hash(tt.password)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, hash)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
				assert.True(t, strings.HasPrefix(hash, "$argon2id$"))

				parts := strings.Split(hash, "$")
				assert.Len(t, parts, 6)
				assert.Equal(t, "argon2id", parts[1])
				assert.Contains(t, parts[3], "m=")
				assert.Contains(t, parts[3], "t=")
				assert.Contains(t, parts[3], "p=")
			}
		})
	}
}

func TestHasher_Verify(t *testing.T) {
	h := NewHasher(DefaultConfig())

	tests := []struct {
		name     string
		password string
		verify   string
		want     bool
		wantErr  bool
	}{
		{
			name:     "verify correct password",
			password: "correctpassword",
			verify:   "correctpassword",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "verify incorrect password",
			password: "correctpassword",
			verify:   "wrongpassword",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "verify empty password",
			password: "",
			verify:   "",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "verify with special characters",
			password: "p@ssw0rd!#$%",
			verify:   "p@ssw0rd!#$%",
			want:     true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := h.Hash(tt.password)
			require.NoError(t, err)

			valid, err := h.Verify(tt.verify, hash)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, valid)
			}
		})
	}
}

func TestHasher_Verify_InvalidHash(t *testing.T) {
	h := NewHasher(DefaultConfig())

	tests := []struct {
		name    string
		hash    string
		wantErr bool
	}{
		{
			name:    "invalid format - too few parts",
			hash:    "$argon2id$v=19",
			wantErr: true,
		},
		{
			name:    "invalid format - no dollar signs",
			hash:    "notavalidhash",
			wantErr: true,
		},
		{
			name:    "invalid version",
			hash:    "$argon2id$v=99$m=65536,t=3,p=2$salt$hash",
			wantErr: true,
		},
		{
			name:    "invalid parameters",
			hash:    "$argon2id$v=19$invalid$ salt$hash",
			wantErr: true,
		},
		{
			name:    "invalid base64 salt",
			hash:    "$argon2id$v=19$m=65536,t=3,p=2$!!!$hash",
			wantErr: true,
		},
		{
			name:    "invalid base64 hash",
			hash:    "$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$!!!",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := h.Verify("password", tt.hash)
			assert.Error(t, err)
			assert.False(t, valid)
		})
	}
}

func TestHasher_DifferentHashesForSamePassword(t *testing.T) {
	h := NewHasher(DefaultConfig())
	password := "samepassword"

	hash1, err := h.Hash(password)
	require.NoError(t, err)

	hash2, err := h.Hash(password)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "two hashes of the same password should be different due to random salt")

	valid1, err := h.Verify(password, hash1)
	require.NoError(t, err)
	assert.True(t, valid1)

	valid2, err := h.Verify(password, hash2)
	require.NoError(t, err)
	assert.True(t, valid2)
}

func TestHasher_VerifyWithDifferentConfig(t *testing.T) {
	config1 := Config{
		Memory:      32768,
		Iterations:  2,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}

	config2 := Config{
		Memory:      65536,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}

	h1 := NewHasher(config1)
	h2 := NewHasher(config2)

	password := "testpassword"

	hash1, err := h1.Hash(password)
	require.NoError(t, err)

	valid, err := h2.Verify(password, hash1)
	require.NoError(t, err)
	assert.True(t, valid, "verification should work with different hasher config")
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, uint32(65536), config.Memory)
	assert.Equal(t, uint32(3), config.Iterations)
	assert.Equal(t, uint8(2), config.Parallelism)
	assert.Equal(t, uint32(16), config.SaltLength)
	assert.Equal(t, uint32(32), config.KeyLength)
}
