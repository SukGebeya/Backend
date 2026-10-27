// Package chapamock stands in for Chapa during development (CHAPA_MODE=mock). It
// serves Chapa's API under /v1, a hosted checkout page where the buyer pays with
// Telebirr, CBE Birr or a test card, and a dashboard. The API talks to it with
// the real chapa.Client, so request and response handling is the same code that
// runs against Chapa. Like Chapa, it sends signed webhooks and calls the callback
// URL when a payment finishes, and transfers settle a little after they're queued.
package chapamock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/SukGebeya/Backend/internal/chapa"
	"github.com/SukGebeya/Backend/pkg/phone"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

const (
	StatusPending   = "pending"
	StatusSuccess   = "success"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
	StatusRefunded  = "refunded"
	StatusReversed  = "reversed"

	MethodTelebirr = "telebirr"
	MethodCBEBirr  = "cbebirr"
	MethodCard     = "card"

	// FailingAccountSuffix makes a transfer fail at the provider, e.g. a Telebirr
	// number that has no wallet, so the payout failure path can be tried.
	FailingAccountSuffix = "0000"
)

type Config struct {
	// PublicURL is where browsers reach the mock, e.g. http://localhost:8080/mock-chapa.
	// The checkout links it returns point there.
	PublicURL string
	// WebhookURL receives payment events, like the URL set in Chapa's dashboard.
	WebhookURL    string
	WebhookSecret string
	// TransferDelay is how long a transfer stays pending before it settles.
	TransferDelay time.Duration
	// Redis keeps state across restarts; nil keeps it in memory.
	Redis redis.Cmdable
}

type Server struct {
	cfg        Config
	store      store
	httpClient *http.Client

	// mu serializes read-modify-write of records; the mock runs in one process.
	mu       sync.Mutex
	inflight sync.WaitGroup
}

type payment struct {
	TxRef       string    `json:"tx_ref"`
	Reference   string    `json:"reference"`
	AmountCents int64     `json:"amount_cents"`
	ChargeCents int64     `json:"charge_cents"`
	Currency    string    `json:"currency"`
	Email       string    `json:"email,omitempty"`
	FirstName   string    `json:"first_name,omitempty"`
	LastName    string    `json:"last_name,omitempty"`
	Phone       string    `json:"phone_number,omitempty"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	CallbackURL string    `json:"callback_url,omitempty"`
	ReturnURL   string    `json:"return_url,omitempty"`
	Status      string    `json:"status"`
	Method      string    `json:"method,omitempty"`
	PayerPhone  string    `json:"payer_phone,omitempty"`
	RefundRef   string    `json:"refund_reference,omitempty"`
	Webhook     string    `json:"webhook,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type transfer struct {
	Reference     string    `json:"reference"`
	ChapaID       string    `json:"chapa_transfer_id"`
	AccountName   string    `json:"account_name"`
	AccountNumber string    `json:"account_number"`
	BankCode      string    `json:"bank_code"`
	BankName      string    `json:"bank_name"`
	Mobile        bool      `json:"mobile"`
	AmountCents   int64     `json:"amount_cents"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	SettleAt      time.Time `json:"settle_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

var banks = []chapa.Bank{
	{ID: "855", Slug: "telebirr", Name: "telebirr", Currency: "ETB", IsMobile: 1},
	{ID: "128", Slug: "cbebirr", Name: "CBEBirr", Currency: "ETB", IsMobile: 1},
	{ID: "946", Slug: "cbe", Name: "Commercial Bank of Ethiopia (CBE)", Currency: "ETB"},
}

func New(cfg Config) *Server {
	cfg.PublicURL = strings.TrimRight(cfg.PublicURL, "/")
	if cfg.TransferDelay <= 0 {
		cfg.TransferDelay = 5 * time.Second
	}
	var st store = newMemoryStore()
	if cfg.Redis != nil {
		st = redisStore{rdb: cfg.Redis}
	}
	return &Server{cfg: cfg, store: st, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

// Handler serves the mock with paths relative to PublicURL.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Route("/v1", func(r chi.Router) {
		r.Use(requireBearer)
		r.Post("/transaction/initialize", s.initialize)
		r.Get("/transaction/verify/{txRef}", s.verify)
		r.Post("/refund/{txRef}", s.refund)
		r.Post("/transfers", s.createTransfer)
		r.Get("/transfers/verify/{reference}", s.verifyTransfer)
		r.Get("/banks", s.listBanks)
	})

	r.Get("/", s.dashboard)
	r.Get("/checkout/{txRef}", s.checkoutPage)
	r.Post("/checkout/{txRef}", s.checkoutSubmit)
	r.Post("/payments/{txRef}/reverse", s.reverse)
	r.Post("/transfers/{reference}/{outcome}", s.settleTransfer)
	return r
}

// Wait blocks until webhook and callback deliveries in flight have finished.
func (s *Server) Wait() { s.inflight.Wait() }

func requireBearer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			apiError(w, http.StatusUnauthorized, "Invalid API Key or User doesn't exist")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) initialize(w http.ResponseWriter, r *http.Request) {
	var req chapa.InitializePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	cents, err := chapa.ParseAmountCents(req.Amount)
	if err != nil || cents <= 0 {
		apiError(w, http.StatusBadRequest, map[string][]string{"amount": {"The amount must be a positive number."}})
		return
	}
	if req.Currency != "ETB" && req.Currency != "USD" {
		apiError(w, http.StatusBadRequest, map[string][]string{"currency": {"The currency must be ETB or USD."}})
		return
	}
	if strings.TrimSpace(req.TxRef) == "" || len(req.TxRef) > 50 {
		apiError(w, http.StatusBadRequest, map[string][]string{"tx_ref": {"The tx_ref is required and must be at most 50 characters."}})
		return
	}
	// Chapa only accepts the 10-digit local form.
	if req.Phone != "" && !isLocalMobile(req.Phone) {
		apiError(w, http.StatusBadRequest, map[string][]string{"phone_number": {"The phone number must be 10 digits, starting with 09 or 07."}})
		return
	}
	if req.Email != "" && !strings.Contains(req.Email, "@") {
		apiError(w, http.StatusBadRequest, map[string][]string{"email": {"The email must be a valid email address."}})
		return
	}

	p := &payment{
		TxRef:       req.TxRef,
		AmountCents: cents,
		Currency:    req.Currency,
		Email:       req.Email,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Phone:       req.Phone,
		CallbackURL: req.CallbackURL,
		ReturnURL:   req.ReturnURL,
		Status:      StatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if req.Customization != nil {
		p.Title, p.Description = req.Customization.Title, req.Customization.Description
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	var existing payment
	found, err := s.store.get(r.Context(), "payment", req.TxRef, &existing)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "Storage error")
		return
	}
	if found {
		apiError(w, http.StatusBadRequest, "Transaction reference has been used before")
		return
	}
	if err := s.store.put(r.Context(), "payment", p.TxRef, p); err != nil {
		apiError(w, http.StatusInternalServerError, "Storage error")
		return
	}

	slog.Info("[MOCK CHAPA] payment initialized", "tx_ref", p.TxRef, "amount", chapa.FormatAmount(cents))
	succeed(w, "Hosted Link", map[string]string{"checkout_url": s.cfg.PublicURL + "/checkout/" + url.PathEscape(p.TxRef)})
}

func (s *Server) verify(w http.ResponseWriter, r *http.Request) {
	p, ok := s.loadPayment(w, r)
	if !ok {
		return
	}
	succeed(w, "Payment details", paymentData(p))
}

// paymentData is the shape Chapa uses in verify responses and webhook payloads.
func paymentData(p *payment) map[string]any {
	data := map[string]any{
		"first_name":     p.FirstName,
		"last_name":      p.LastName,
		"email":          p.Email,
		"mobile":         p.PayerPhone,
		"currency":       p.Currency,
		"amount":         json.Number(chapa.FormatAmount(p.AmountCents)),
		"charge":         json.Number(chapa.FormatAmount(p.ChargeCents)),
		"mode":           chapa.ModeTest,
		"method":         p.Method,
		"payment_method": p.Method,
		"type":           "API",
		"status":         p.Status,
		"reference":      p.Reference,
		"tx_ref":         p.TxRef,
		"customization":  map[string]any{"title": p.Title, "description": p.Description, "logo": nil},
		"meta":           nil,
		"created_at":     p.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":     p.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if p.Status == StatusPending {
		data["charge"] = nil
	}
	return data
}

func (s *Server) refund(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason    string `json:"reason"`
		Reference string `json:"reference"`
		Amount    string `json:"amount"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.loadPayment(w, r)
	if !ok {
		return
	}

	switch {
	case p.Status == StatusRefunded && req.Reference != "" && req.Reference == p.RefundRef:
		// A retry of the same refund; Chapa would otherwise refuse a second one.
		succeed(w, "Refund processed successfully", nil)
		return
	case p.Status != StatusSuccess:
		apiError(w, http.StatusBadRequest, "Transaction is not eligible for refund")
		return
	}
	if req.Amount != "" {
		if cents, err := chapa.ParseAmountCents(req.Amount); err != nil || cents <= 0 || cents > p.AmountCents {
			apiError(w, http.StatusBadRequest, "Refund amount is invalid")
			return
		}
	}

	p.Status, p.RefundRef, p.UpdatedAt = StatusRefunded, req.Reference, time.Now()
	if err := s.store.put(r.Context(), "payment", p.TxRef, p); err != nil {
		apiError(w, http.StatusInternalServerError, "Storage error")
		return
	}
	slog.Info("[MOCK CHAPA] refund processed", "tx_ref", p.TxRef, "reason", req.Reason)
	s.notify(*p, "charge.refunded")
	succeed(w, "Refund processed successfully", nil)
}

func (s *Server) createTransfer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountName   string `json:"account_name"`
		AccountNumber string `json:"account_number"`
		Amount        string `json:"amount"`
		Currency      string `json:"currency"`
		Reference     string `json:"reference"`
		BankCode      string `json:"bank_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	bank, ok := findBank(req.BankCode)
	if !ok {
		apiError(w, http.StatusBadRequest, map[string][]string{"bank_code": {"The selected bank code is invalid."}})
		return
	}
	cents, err := chapa.ParseAmountCents(req.Amount)
	if err != nil || cents <= 0 {
		apiError(w, http.StatusBadRequest, map[string][]string{"amount": {"The amount must be a positive number."}})
		return
	}
	if strings.TrimSpace(req.AccountName) == "" {
		apiError(w, http.StatusBadRequest, map[string][]string{"account_name": {"The account name is required."}})
		return
	}
	if strings.TrimSpace(req.Reference) == "" {
		apiError(w, http.StatusBadRequest, map[string][]string{"reference": {"The reference is required."}})
		return
	}
	mobile := bank.IsMobile == 1
	if mobile && !isLocalMobile(req.AccountNumber) {
		apiError(w, http.StatusBadRequest, map[string][]string{"account_number": {"Mobile money accounts must be 10 digits, starting with 09 or 07."}})
		return
	}
	if !mobile && !isDigits(req.AccountNumber, 8, 16) {
		apiError(w, http.StatusBadRequest, map[string][]string{"account_number": {"The account number must be 8 to 16 digits."}})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	var existing transfer
	found, err := s.store.get(r.Context(), "transfer", req.Reference, &existing)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "Storage error")
		return
	}
	if found {
		if existing.AmountCents != cents || existing.AccountNumber != req.AccountNumber {
			apiError(w, http.StatusBadRequest, "Transfer reference has been used before")
			return
		}
	} else {
		now := time.Now()
		existing = transfer{
			Reference:     req.Reference,
			ChapaID:       randomRef("TR"),
			AccountName:   req.AccountName,
			AccountNumber: req.AccountNumber,
			BankCode:      bank.ID.String(),
			BankName:      bank.Name,
			Mobile:        mobile,
			AmountCents:   cents,
			Currency:      req.Currency,
			Status:        StatusPending,
			SettleAt:      now.Add(s.cfg.TransferDelay),
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := s.store.put(r.Context(), "transfer", existing.Reference, &existing); err != nil {
			apiError(w, http.StatusInternalServerError, "Storage error")
			return
		}
		slog.Info("[MOCK CHAPA] transfer queued", "reference", existing.Reference, "bank", bank.Name, "account", existing.AccountNumber, "amount", chapa.FormatAmount(cents))
	}

	succeed(w, "Transfer Queued Successfully", existing.Reference)
}

func (s *Server) verifyTransfer(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.loadTransfer(w, r, chi.URLParam(r, "reference"))
	if !ok {
		return
	}
	if err := s.settleIfDue(r.Context(), t); err != nil {
		apiError(w, http.StatusInternalServerError, "Storage error")
		return
	}

	method := "bank"
	if t.Mobile {
		method = "mobile_money"
	}
	succeed(w, "Transfer details", map[string]any{
		"account_name":      t.AccountName,
		"account_number":    t.AccountNumber,
		"currency":          t.Currency,
		"amount":            json.Number(chapa.FormatAmount(t.AmountCents)),
		"charge":            json.Number("0"),
		"mode":              chapa.ModeTest,
		"transfer_method":   method,
		"chapa_transfer_id": t.ChapaID,
		"bank_code":         json.Number(t.BankCode),
		"bank_name":         t.BankName,
		"status":            t.Status,
		"tx_ref":            t.Reference,
		"created_at":        t.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":        t.UpdatedAt.UTC().Format(time.RFC3339),
	})
}

// settleIfDue moves a pending transfer to its outcome once its delay has passed.
func (s *Server) settleIfDue(ctx context.Context, t *transfer) error {
	if t.Status != StatusPending || time.Now().Before(t.SettleAt) {
		return nil
	}
	t.Status = StatusSuccess
	if strings.HasSuffix(t.AccountNumber, FailingAccountSuffix) {
		t.Status = StatusFailed
	}
	t.UpdatedAt = time.Now()
	slog.Info("[MOCK CHAPA] transfer settled", "reference", t.Reference, "status", t.Status)
	return s.store.put(ctx, "transfer", t.Reference, t)
}

func (s *Server) listBanks(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, envelope{Message: "Banks retrieved", Data: banks})
}

func (s *Server) loadPayment(w http.ResponseWriter, r *http.Request) (*payment, bool) {
	var p payment
	found, err := s.store.get(r.Context(), "payment", chi.URLParam(r, "txRef"), &p)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "Storage error")
		return nil, false
	}
	if !found {
		apiError(w, http.StatusNotFound, "Invalid transaction or Transaction not found")
		return nil, false
	}
	return &p, true
}

func (s *Server) loadTransfer(w http.ResponseWriter, r *http.Request, reference string) (*transfer, bool) {
	var t transfer
	found, err := s.store.get(r.Context(), "transfer", reference, &t)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "Storage error")
		return nil, false
	}
	if !found {
		apiError(w, http.StatusNotFound, "Transfer not found")
		return nil, false
	}
	return &t, true
}

func findBank(code string) (chapa.Bank, bool) {
	for _, b := range banks {
		if b.ID.String() == strings.TrimSpace(code) {
			return b, true
		}
	}
	return chapa.Bank{}, false
}

func isLocalMobile(number string) bool {
	local, err := phone.Local(number)
	return err == nil && local == number
}

func isDigits(s string, minLen, maxLen int) bool {
	if len(s) < minLen || len(s) > maxLen {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func randomRef(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return prefix + "mock" + hex.EncodeToString(b)
}

// envelope is the shape of every Chapa API response. Chapa leaves status out
// of the bank list.
type envelope struct {
	Message any    `json:"message"`
	Status  string `json:"status,omitempty"`
	Data    any    `json:"data"`
}

func succeed(w http.ResponseWriter, message string, data any) {
	writeJSON(w, http.StatusOK, envelope{Message: message, Status: chapa.StatusSuccess, Data: data})
}

func apiError(w http.ResponseWriter, status int, message any) {
	writeJSON(w, status, envelope{Message: message, Status: chapa.StatusFailed})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("[MOCK CHAPA] failed to write response", "error", err)
	}
}

func formatETB(cents int64) string {
	whole := fmt.Sprintf("%d", cents/100)
	var b strings.Builder
	for i, d := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(d)
	}
	return fmt.Sprintf("%s.%02d ETB", b.String(), cents%100)
}
