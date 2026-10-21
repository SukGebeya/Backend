package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/SukGebeya/Backend/internal/chapa"
	"github.com/SukGebeya/Backend/internal/jobs"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	code := m.Run()
	testutil.StopSharedPostgres()
	os.Exit(code)
}

func TestHandleChapa(t *testing.T) {
	pool := testutil.NewTestDatabase(t)
	enqueuer := &jobs.NoopEnqueuer{}
	secret := "whsec_test"
	h := NewWebhookHandler(
		chapa.NewFakeClient(chapa.Config{WebhookSecret: secret}),
		repository.NewWebhookEventRepository(pool),
		repository.NewTransactionManager(pool),
		enqueuer,
	)

	send := func(body string, headers map[string]string) int {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/chapa", strings.NewReader(body))
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		h.HandleChapa(rec, req)
		return rec.Code
	}

	body := `{"event":"charge.success","tx_ref":"GEBEYA-abc-1","status":"success","amount":"100.00"}`

	require.Equal(t, http.StatusUnauthorized, send(body, nil))
	require.Equal(t, http.StatusUnauthorized, send(body, map[string]string{"x-chapa-signature": chapa.Sign("wrong", []byte(body))}))
	require.Empty(t, enqueuer.Jobs)

	require.Equal(t, http.StatusOK, send(body, map[string]string{"x-chapa-signature": chapa.Sign(secret, []byte(body))}))
	require.Len(t, enqueuer.Jobs, 1)
	args, ok := enqueuer.Jobs[0].(jobs.ProcessPaymentWebhookArgs)
	require.True(t, ok)
	require.Equal(t, "GEBEYA-abc-1", args.TxRef)
	require.NotNil(t, args.WebhookEventID)

	require.Equal(t, http.StatusOK, send(body, map[string]string{"Chapa-Signature": chapa.Sign(secret, []byte(secret))}))
	require.Len(t, enqueuer.Jobs, 1, "a replayed delivery must not enqueue a second job")

	other := `{"event":"payout.success","reference":"x"}`
	require.Equal(t, http.StatusOK, send(other, map[string]string{"x-chapa-signature": chapa.Sign(secret, []byte(other))}))
	require.Len(t, enqueuer.Jobs, 1, "non-payment events are recorded but not processed")
}

func TestHandleChapaReversalNeedsPayloadSignature(t *testing.T) {
	pool := testutil.NewTestDatabase(t)
	enqueuer := &jobs.NoopEnqueuer{}
	secret := "whsec_test"
	h := NewWebhookHandler(
		chapa.NewFakeClient(chapa.Config{WebhookSecret: secret}),
		repository.NewWebhookEventRepository(pool),
		repository.NewTransactionManager(pool),
		enqueuer,
	)

	send := func(body string, headers map[string]string) int {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/chapa", strings.NewReader(body))
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		h.HandleChapa(rec, req)
		return rec.Code
	}

	weak := `{"event":"charge.reversed","tx_ref":"GB-abc-1","status":"reversed"}`
	require.Equal(t, http.StatusOK, send(weak, map[string]string{"Chapa-Signature": chapa.Sign(secret, []byte(secret))}))
	require.Empty(t, enqueuer.Jobs, "a reversal with only the static header is recorded for review, not acted on")

	strong := `{"event":"charge.reversed","tx_ref":"GB-abc-2","status":"reversed"}`
	require.Equal(t, http.StatusOK, send(strong, map[string]string{"x-chapa-signature": chapa.Sign(secret, []byte(strong))}))
	require.Len(t, enqueuer.Jobs, 1)
	args, ok := enqueuer.Jobs[0].(jobs.ProcessPaymentReversalArgs)
	require.True(t, ok)
	require.Equal(t, "GB-abc-2", args.TxRef)
}
