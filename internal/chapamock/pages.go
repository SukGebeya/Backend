package chapamock

import (
	"context"
	"embed"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/SukGebeya/Backend/internal/chapa"
	"github.com/SukGebeya/Backend/pkg/phone"
	"github.com/go-chi/chi/v5"
)

//go:embed pages.html
var pageFiles embed.FS

var errNotEligible = errors.New("not eligible")

var pages = template.Must(template.New("").Funcs(template.FuncMap{
	"etb":  formatETB,
	"time": func(t time.Time) string { return t.Local().Format("2 Jan 15:04:05") },
}).ParseFS(pageFiles, "pages.html"))

type checkoutView struct {
	Payment *payment
	Base    string
	Method  string
	Phone   string
	Error   string
}

type dashboardView struct {
	Base          string
	Payments      []*payment
	Transfers     []*transfer
	FailingSuffix string
	TransferDelay time.Duration
}

var paymentMethods = map[string]string{
	MethodTelebirr: "Telebirr",
	MethodCBEBirr:  "CBE Birr",
	MethodCard:     "Card",
}

func (s *Server) checkoutPage(w http.ResponseWriter, r *http.Request) {
	var p payment
	found, err := s.store.get(r.Context(), "payment", chi.URLParam(r, "txRef"), &p)
	if err != nil {
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "This payment link is not valid.", http.StatusNotFound)
		return
	}
	s.render(w, http.StatusOK, "checkout", checkoutView{Payment: &p, Base: s.cfg.PublicURL, Method: MethodTelebirr, Phone: p.Phone})
}

func (s *Server) checkoutSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	action := r.PostFormValue("action")
	method := r.PostFormValue("method")
	payer := strings.TrimSpace(r.PostFormValue("phone"))

	p, view, status, event := s.applyCheckout(r.Context(), chi.URLParam(r, "txRef"), action, method, payer)
	if p == nil {
		http.Error(w, view.Error, status)
		return
	}
	if event == "" {
		s.render(w, status, "checkout", view)
		return
	}

	s.notify(*p, event)
	if p.ReturnURL != "" {
		http.Redirect(w, r, p.ReturnURL, http.StatusSeeOther)
		return
	}
	s.render(w, http.StatusOK, "checkout", view)
}

// applyCheckout records the buyer's choice. It returns the event to send, or an
// empty event when the page should be shown again (an error, or a payment that
// already finished).
func (s *Server) applyCheckout(ctx context.Context, txRef, action, method, payer string) (*payment, checkoutView, int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var p payment
	found, err := s.store.get(ctx, "payment", txRef, &p)
	if err != nil {
		return nil, checkoutView{Error: "storage error"}, http.StatusInternalServerError, ""
	}
	if !found {
		return nil, checkoutView{Error: "This payment link is not valid."}, http.StatusNotFound, ""
	}
	view := checkoutView{Payment: &p, Base: s.cfg.PublicURL, Method: method, Phone: payer}
	if p.Status != StatusPending {
		return &p, view, http.StatusOK, ""
	}
	if _, ok := paymentMethods[method]; !ok && action != "cancel" {
		view.Method, view.Error = MethodTelebirr, "Choose how to pay."
		return &p, view, http.StatusUnprocessableEntity, ""
	}

	var event string
	switch action {
	case "pay":
		if method == MethodCard {
			p.PayerPhone = p.Phone
		} else {
			local, err := phone.Local(payer)
			if err != nil {
				view.Error = "Enter the " + paymentMethods[method] + " number, like 0911 234 567."
				return &p, view, http.StatusUnprocessableEntity, ""
			}
			p.PayerPhone = local
		}
		p.Method, p.Status, p.ChargeCents = method, StatusSuccess, chapa.FakeChargeCents(p.AmountCents)
		event = "charge.success"
	case "decline":
		p.Method, p.Status = method, StatusFailed
		event = "charge.failed"
	case "cancel":
		p.Status = StatusCancelled
		event = "charge.cancelled"
	default:
		view.Error = "Unknown action."
		return &p, view, http.StatusBadRequest, ""
	}

	p.Reference, p.UpdatedAt = randomRef("AP"), time.Now()
	if err := s.store.put(ctx, "payment", p.TxRef, &p); err != nil {
		return nil, checkoutView{Error: "storage error"}, http.StatusInternalServerError, ""
	}
	slog.Info("[MOCK CHAPA] checkout finished", "tx_ref", p.TxRef, "status", p.Status, "method", p.Method)
	return &p, view, http.StatusOK, event
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	view := dashboardView{Base: s.cfg.PublicURL, FailingSuffix: FailingAccountSuffix, TransferDelay: s.cfg.TransferDelay}

	s.mu.Lock()
	defer s.mu.Unlock()
	ids, err := s.store.recent(ctx, "payment", 50)
	if err == nil {
		for _, id := range ids {
			var p payment
			if found, err := s.store.get(ctx, "payment", id, &p); err == nil && found {
				view.Payments = append(view.Payments, &p)
			}
		}
	}
	ids, err = s.store.recent(ctx, "transfer", 50)
	if err == nil {
		for _, id := range ids {
			var t transfer
			if found, err := s.store.get(ctx, "transfer", id, &t); err == nil && found {
				_ = s.settleIfDue(ctx, &t)
				view.Transfers = append(view.Transfers, &t)
			}
		}
	}
	s.render(w, http.StatusOK, "dashboard", view)
}

// reverse simulates a chargeback, or a refund issued from Chapa's dashboard.
func (s *Server) reverse(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	var p payment
	found, err := s.store.get(r.Context(), "payment", chi.URLParam(r, "txRef"), &p)
	if err == nil && found && p.Status == StatusSuccess {
		p.Status, p.UpdatedAt = StatusReversed, time.Now()
		err = s.store.put(r.Context(), "payment", p.TxRef, &p)
	} else if err == nil {
		err = errNotEligible
	}
	s.mu.Unlock()
	if err != nil {
		http.Error(w, "Only successful payments can be reversed.", http.StatusBadRequest)
		return
	}
	slog.Info("[MOCK CHAPA] payment reversed", "tx_ref", p.TxRef)
	s.notify(p, "charge.reversed")
	http.Redirect(w, r, s.cfg.PublicURL+"/", http.StatusSeeOther)
}

// settleTransfer settles a pending transfer now instead of after TransferDelay.
func (s *Server) settleTransfer(w http.ResponseWriter, r *http.Request) {
	outcome := map[string]string{"sent": StatusSuccess, "failed": StatusFailed}[chi.URLParam(r, "outcome")]
	s.mu.Lock()
	var t transfer
	found, err := s.store.get(r.Context(), "transfer", chi.URLParam(r, "reference"), &t)
	if err == nil && found && t.Status == StatusPending && outcome != "" {
		t.Status, t.UpdatedAt = outcome, time.Now()
		err = s.store.put(r.Context(), "transfer", t.Reference, &t)
	} else if err == nil {
		err = errNotEligible
	}
	s.mu.Unlock()
	if err != nil {
		http.Error(w, "Only pending transfers can be settled.", http.StatusBadRequest)
		return
	}
	slog.Info("[MOCK CHAPA] transfer settled by hand", "reference", t.Reference, "status", t.Status)
	http.Redirect(w, r, s.cfg.PublicURL+"/", http.StatusSeeOther)
}

func (s *Server) render(w http.ResponseWriter, status int, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := pages.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("[MOCK CHAPA] failed to render page", "page", name, "error", err)
	}
}
