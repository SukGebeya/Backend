package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	code := m.Run()
	testutil.StopSharedRedis()
	os.Exit(code)
}

func TestIdempotency(t *testing.T) {
	repo := repository.NewIdempotencyRepository(testutil.NewTestRedis(t))

	calls := 0
	handler := Idempotency(repo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"order_id":"` + uuid.NewString() + `"}`))
	}))

	alice, bob := uuid.New(), uuid.New()
	do := func(user uuid.UUID) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/orders/checkout", nil)
		req.Header.Set(IdempotencyKeyHeader, "key-1")
		req = req.WithContext(context.WithValue(req.Context(), UserIDKey, user))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	first := do(alice)
	require.Equal(t, http.StatusCreated, first.Code)

	replay := do(alice)
	require.Equal(t, http.StatusCreated, replay.Code, "replay keeps the original status")
	require.Equal(t, "true", replay.Header().Get("X-Idempotent-Replayed"))
	require.Equal(t, first.Body.String(), replay.Body.String())
	require.Equal(t, 1, calls)

	other := do(bob)
	require.Equal(t, http.StatusCreated, other.Code)
	require.Empty(t, other.Header().Get("X-Idempotent-Replayed"), "keys are scoped per user")
	require.Equal(t, 2, calls)

	carol := uuid.New()
	hash := sha256.Sum256([]byte(carol.String() + ":POST:/api/v1/orders/checkout:key-1"))
	locked, err := repo.Lock(context.Background(), "idempotency:"+hex.EncodeToString(hash[:])+":lock", 60)
	require.NoError(t, err)
	require.True(t, locked)

	inFlight := do(carol)
	require.Equal(t, http.StatusConflict, inFlight.Code)
	require.Equal(t, 2, calls)
}
