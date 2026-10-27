package chapamock

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SukGebeya/Backend/internal/chapa"
)

const secret = "dev-webhook-secret"

type received struct {
	mu        sync.Mutex
	webhooks  []map[string]any
	callbacks []url.Values
}

func (rc *received) events() []string {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	out := make([]string, 0, len(rc.webhooks))
	for _, w := range rc.webhooks {
		out = append(out, w["event"].(string))
	}
	return out
}

type env struct {
	mock   *Server
	client *chapa.Client
	url    string
	got    *received
}

// newEnv runs the mock, a receiver for its webhooks and callbacks, and the real
// Chapa client pointed at the mock.
func newEnv(t *testing.T) *env {
	t.Helper()
	got := &received{}
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.mu.Lock()
		defer got.mu.Unlock()
		if r.Method == http.MethodGet {
			got.callbacks = append(got.callbacks, r.URL.Query())
			return
		}
		body, _ := io.ReadAll(r.Body)
		if !chapa.VerifyPayload(secret, body, r.Header.Get("x-chapa-signature")) ||
			!chapa.VerifySignature(secret, body, r.Header.Get("Chapa-Signature")) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		got.webhooks = append(got.webhooks, payload)
	}))
	t.Cleanup(receiver.Close)

	var handler http.Handler
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handler.ServeHTTP(w, r) }))
	t.Cleanup(srv.Close)
	mock := New(Config{PublicURL: srv.URL, WebhookURL: receiver.URL + "/webhook", WebhookSecret: secret, TransferDelay: 50 * time.Millisecond})
	handler = mock.Handler()

	client := chapa.NewClient(chapa.Config{
		BaseURL:       srv.URL + "/v1",
		SecretKey:     "CHASECK_TEST-mock",
		WebhookSecret: secret,
		CallbackURL:   receiver.URL + "/callback",
		ReturnURL:     "http://shop.test/payment/complete",
	})
	return &env{mock: mock, client: client, url: srv.URL, got: got}
}

func (e *env) initialize(t *testing.T, txRef string) string {
	t.Helper()
	resp, err := e.client.InitializePayment(context.Background(), chapa.InitializePaymentRequest{
		Amount: "2650.00", Currency: "ETB", TxRef: txRef, Phone: "0911234567", FirstName: "Abebe", LastName: "Bikila",
		Customization: &chapa.Customization{Title: "Gebeya", Description: "Order 3eb05c7c"},
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	return resp.Data.CheckoutURL
}

type result struct {
	code     int
	location string
	body     string
}

var noRedirect = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

// send makes a request like the buyer's browser would, without following the
// redirect, so the test sees where it points.
func send(t *testing.T, method, target string, form url.Values) result {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, target, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := noRedirect.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	return result{code: resp.StatusCode, location: resp.Header.Get("Location"), body: string(body)}
}

// submit posts the hosted checkout form.
func (e *env) submit(t *testing.T, checkoutURL string, form url.Values) result {
	t.Helper()
	return send(t, http.MethodPost, checkoutURL, form)
}

func (e *env) status(t *testing.T, txRef string) string {
	t.Helper()
	resp, err := e.client.VerifyPayment(context.Background(), txRef)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	return resp.Data.Status
}

func TestPaymentWithTelebirr(t *testing.T) {
	e := newEnv(t)
	checkoutURL := e.initialize(t, "GEBEYA-1")
	if checkoutURL != e.url+"/checkout/GEBEYA-1" {
		t.Fatalf("checkout URL = %q", checkoutURL)
	}
	if got := e.status(t, "GEBEYA-1"); got != StatusPending {
		t.Fatalf("status before paying = %q", got)
	}

	page := send(t, http.MethodGet, checkoutURL, nil)
	if !strings.Contains(page.body, "Pay 2,650.00 ETB") || !strings.Contains(page.body, `value="0911234567"`) {
		t.Fatalf("checkout page lacks the amount or the buyer's phone:\n%s", page.body)
	}

	resp := e.submit(t, checkoutURL, url.Values{"action": {"pay"}, "method": {"telebirr"}, "phone": {"+251 91 123 4567"}})
	if resp.code != http.StatusSeeOther || resp.location != "http://shop.test/payment/complete" {
		t.Fatalf("pay: %d to %q, want 303 to the return URL", resp.code, resp.location)
	}

	verified, err := e.client.VerifyPayment(context.Background(), "GEBEYA-1")
	if err != nil {
		t.Fatal(err)
	}
	if verified.Data.Status != StatusSuccess || verified.Data.Amount.String() != "2650.00" || verified.Data.Charge.String() != "92.75" {
		t.Fatalf("verify after paying = %+v", verified.Data)
	}

	e.mock.Wait()
	if got := e.got.events(); len(got) != 1 || got[0] != "charge.success" {
		t.Fatalf("webhooks = %v, want one signed charge.success", got)
	}
	if hook := e.got.webhooks[0]; hook["tx_ref"] != "GEBEYA-1" || hook["payment_method"] != MethodTelebirr || hook["mobile"] != "0911234567" {
		t.Errorf("webhook payload = %v", hook)
	}
	if len(e.got.callbacks) != 1 || e.got.callbacks[0].Get("trx_ref") != "GEBEYA-1" || e.got.callbacks[0].Get("status") != StatusSuccess {
		t.Errorf("callbacks = %v", e.got.callbacks)
	}

	// The page now shows the result, and paying again changes nothing.
	again := e.submit(t, checkoutURL, url.Values{"action": {"decline"}, "method": {"telebirr"}})
	if again.code != http.StatusOK || e.status(t, "GEBEYA-1") != StatusSuccess {
		t.Errorf("a finished payment changed on resubmit")
	}
}

func TestDeclineAndCancel(t *testing.T) {
	e := newEnv(t)
	declined := e.initialize(t, "GEBEYA-2")
	cancelled := e.initialize(t, "GEBEYA-3")

	e.submit(t, declined, url.Values{"action": {"decline"}, "method": {"cbebirr"}})
	e.submit(t, cancelled, url.Values{"action": {"cancel"}})

	if got := e.status(t, "GEBEYA-2"); got != StatusFailed {
		t.Errorf("declined payment status = %q", got)
	}
	if got := e.status(t, "GEBEYA-3"); got != StatusCancelled {
		t.Errorf("cancelled payment status = %q", got)
	}
	e.mock.Wait()
	if got := strings.Join(e.got.events(), ","); !strings.Contains(got, "charge.failed") || !strings.Contains(got, "charge.cancelled") {
		t.Errorf("webhooks = %v", got)
	}
	if len(e.got.callbacks) != 0 {
		t.Errorf("callback called for unsuccessful payments: %v", e.got.callbacks)
	}
}

func TestPayNeedsTheMobileMoneyNumber(t *testing.T) {
	e := newEnv(t)
	checkoutURL := e.initialize(t, "GEBEYA-4")

	resp := e.submit(t, checkoutURL, url.Values{"action": {"pay"}, "method": {"telebirr"}, "phone": {"12345"}})
	if resp.code != http.StatusUnprocessableEntity {
		t.Fatalf("pay with a bad number: %d, want 422", resp.code)
	}
	if got := e.status(t, "GEBEYA-4"); got != StatusPending {
		t.Errorf("status = %q, want still pending", got)
	}

	resp = e.submit(t, checkoutURL, url.Values{"action": {"pay"}, "method": {"card"}})
	if resp.code != http.StatusSeeOther || e.status(t, "GEBEYA-4") != StatusSuccess {
		t.Errorf("card payment did not succeed (%d)", resp.code)
	}
}

func TestInitializeValidatesLikeChapa(t *testing.T) {
	e := newEnv(t)
	e.initialize(t, "GEBEYA-5")

	cases := map[string]chapa.InitializePaymentRequest{
		"duplicate tx_ref": {Amount: "10.00", Currency: "ETB", TxRef: "GEBEYA-5"},
		"E.164 phone":      {Amount: "10.00", Currency: "ETB", TxRef: "GEBEYA-6", Phone: "+251911234567"},
		"zero amount":      {Amount: "0", Currency: "ETB", TxRef: "GEBEYA-7"},
		"bad currency":     {Amount: "10.00", Currency: "EUR", TxRef: "GEBEYA-8"},
	}
	for name, req := range cases {
		_, err := e.client.InitializePayment(context.Background(), req)
		var apiErr *chapa.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: err = %v, want a 400 from the API", name, err)
		}
	}

	if _, err := e.client.VerifyPayment(context.Background(), "UNKNOWN"); !isStatus(err, http.StatusNotFound) {
		t.Errorf("verify unknown tx_ref: %v, want 404", err)
	}
}

func TestRefundAndReversal(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	if _, err := e.client.Refund(ctx, "GEBEYA-9", chapa.RefundRequest{Reference: "refund-1"}); !isStatus(err, http.StatusNotFound) {
		t.Errorf("refund unknown payment: %v, want 404", err)
	}
	e.submit(t, e.initialize(t, "GEBEYA-9"), url.Values{"action": {"pay"}, "method": {"card"}})
	if _, err := e.client.Refund(ctx, "GEBEYA-9", chapa.RefundRequest{Reason: "Order cancelled", Reference: "refund-1"}); err != nil {
		t.Fatalf("refund: %v", err)
	}
	if _, err := e.client.Refund(ctx, "GEBEYA-9", chapa.RefundRequest{Reference: "refund-1"}); err != nil {
		t.Errorf("retrying the same refund: %v", err)
	}
	if _, err := e.client.Refund(ctx, "GEBEYA-9", chapa.RefundRequest{Reference: "refund-2"}); !isStatus(err, http.StatusBadRequest) {
		t.Errorf("second refund: %v, want 400", err)
	}
	if got := e.status(t, "GEBEYA-9"); got != StatusRefunded {
		t.Errorf("status after refund = %q", got)
	}

	e.submit(t, e.initialize(t, "GEBEYA-10"), url.Values{"action": {"pay"}, "method": {"card"}})
	if got := send(t, http.MethodPost, e.url+"/payments/GEBEYA-10/reverse", nil); got.code != http.StatusSeeOther {
		t.Fatalf("reverse: %d %s", got.code, got.body)
	}
	if got := e.status(t, "GEBEYA-10"); got != StatusReversed {
		t.Errorf("status after reversal = %q", got)
	}

	e.mock.Wait()
	events := strings.Join(e.got.events(), ",")
	if !strings.Contains(events, "charge.refunded") || !strings.Contains(events, "charge.reversed") {
		t.Errorf("webhooks = %v, want charge.refunded and charge.reversed", events)
	}
}

func TestTransfers(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	telebirr := chapa.TransferRequest{AccountName: "Tigist Bekele", AccountNumber: "0911234567", AmountCents: 100000, Currency: "ETB", Reference: "GEBEYA-PO-1", BankCode: "855"}
	if _, err := e.client.InitiateTransfer(ctx, telebirr); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if _, err := e.client.InitiateTransfer(ctx, telebirr); err != nil {
		t.Errorf("repeating the same transfer: %v", err)
	}
	if got := transferStatus(t, e, "GEBEYA-PO-1"); got != StatusPending {
		t.Errorf("status right after queueing = %q", got)
	}

	failing := telebirr
	failing.Reference, failing.AccountNumber = "GEBEYA-PO-2", "0911110000"
	if _, err := e.client.InitiateTransfer(ctx, failing); err != nil {
		t.Fatalf("transfer to the failing number: %v", err)
	}

	time.Sleep(80 * time.Millisecond)
	if got := transferStatus(t, e, "GEBEYA-PO-1"); got != StatusSuccess {
		t.Errorf("status after the delay = %q, want success", got)
	}
	if got := transferStatus(t, e, "GEBEYA-PO-2"); got != StatusFailed {
		t.Errorf("failing number status = %q, want failed", got)
	}

	rejected := map[string]chapa.TransferRequest{
		"unknown bank":           {AccountName: "A", AccountNumber: "0911234567", AmountCents: 100, Currency: "ETB", Reference: "R1", BankCode: "1"},
		"telebirr not a phone":   {AccountName: "A", AccountNumber: "12345", AmountCents: 100, Currency: "ETB", Reference: "R2", BankCode: "855"},
		"bank account too short": {AccountName: "A", AccountNumber: "123", AmountCents: 100, Currency: "ETB", Reference: "R3", BankCode: "946"},
		"reused reference":       {AccountName: "A", AccountNumber: "0922222222", AmountCents: 100, Currency: "ETB", Reference: "GEBEYA-PO-1", BankCode: "855"},
	}
	for name, req := range rejected {
		if _, err := e.client.InitiateTransfer(ctx, req); !isStatus(err, http.StatusBadRequest) {
			t.Errorf("%s: %v, want 400", name, err)
		}
	}
	if _, err := e.client.VerifyTransfer(ctx, "NOPE"); !isStatus(err, http.StatusNotFound) {
		t.Errorf("verify unknown transfer: %v, want 404", err)
	}

	banks, err := e.client.ListBanks(ctx)
	if err != nil || len(banks) == 0 || banks[0].Slug != "telebirr" {
		t.Errorf("banks = %v, %v", banks, err)
	}
}

func TestRequiresAnAPIKey(t *testing.T) {
	e := newEnv(t)
	if got := send(t, http.MethodGet, e.url+"/v1/banks", nil); got.code != http.StatusUnauthorized {
		t.Errorf("request without a key: %d, want 401", got.code)
	}
}

func TestMemoryStoreListsNewestFirst(t *testing.T) {
	s := newMemoryStore()
	ctx := context.Background()
	for _, id := range []string{"a", "b", "c"} {
		_ = s.put(ctx, "payment", id, map[string]string{"id": id})
		time.Sleep(time.Millisecond)
	}
	_ = s.put(ctx, "payment", "a", map[string]string{"id": "a", "status": "success"})
	ids, _ := s.recent(ctx, "payment", 2)
	if strings.Join(ids, ",") != "c,b" {
		t.Errorf("recent = %v, want c,b (updates keep their place)", ids)
	}
}

func transferStatus(t *testing.T, e *env, reference string) string {
	t.Helper()
	resp, err := e.client.VerifyTransfer(context.Background(), reference)
	if err != nil {
		t.Fatalf("verify transfer: %v", err)
	}
	return resp.Data.Status
}

func isStatus(err error, code int) bool {
	var apiErr *chapa.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == code
}
