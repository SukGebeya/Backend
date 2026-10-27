package chapamock

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/SukGebeya/Backend/internal/chapa"
)

var webhookRetryDelays = []time.Duration{0, time.Second, 3 * time.Second}

// notify delivers a payment event in the background, like Chapa: a signed webhook
// to WebhookURL and, for successful payments, a GET to the payment's callback URL.
func (s *Server) notify(p payment, event string) {
	s.inflight.Add(1)
	go func() {
		defer s.inflight.Done()
		result := s.deliverWebhook(p, event)
		if event == "charge.success" && p.CallbackURL != "" {
			s.callback(p)
		}

		s.mu.Lock()
		defer s.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var current payment
		if found, err := s.store.get(ctx, "payment", p.TxRef, &current); err == nil && found {
			current.Webhook = event + ": " + result
			_ = s.store.put(ctx, "payment", current.TxRef, &current)
		}
	}()
}

func (s *Server) deliverWebhook(p payment, event string) string {
	if s.cfg.WebhookURL == "" {
		return "no webhook URL"
	}
	payload := paymentData(&p)
	payload["event"] = event
	body, err := json.Marshal(payload)
	if err != nil {
		return err.Error()
	}

	result := ""
	for _, delay := range webhookRetryDelays {
		time.Sleep(delay)
		code, err := s.postWebhook(body)
		if err == nil && code >= 200 && code < 300 {
			slog.Info("[MOCK CHAPA] webhook delivered", "event", event, "tx_ref", p.TxRef, "status_code", code)
			return fmt.Sprintf("delivered (%d)", code)
		}
		if err != nil {
			result = err.Error()
		} else {
			result = fmt.Sprintf("HTTP %d", code)
		}
	}
	slog.Warn("[MOCK CHAPA] webhook not delivered", "event", event, "tx_ref", p.TxRef, "result", result)
	return "failed: " + result
}

func (s *Server) postWebhook(body []byte) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-chapa-signature", chapa.Sign(s.cfg.WebhookSecret, body))
	req.Header.Set("Chapa-Signature", chapa.Sign(s.cfg.WebhookSecret, []byte(s.cfg.WebhookSecret)))
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	_ = resp.Body.Close()
	return resp.StatusCode, nil
}

func (s *Server) callback(p payment) {
	target, err := url.Parse(p.CallbackURL)
	if err != nil {
		slog.Warn("[MOCK CHAPA] invalid callback URL", "tx_ref", p.TxRef, "callback_url", p.CallbackURL)
		return
	}
	q := target.Query()
	q.Set("trx_ref", p.TxRef)
	q.Set("ref_id", p.Reference)
	q.Set("status", p.Status)
	target.RawQuery = q.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		slog.Warn("[MOCK CHAPA] callback failed", "tx_ref", p.TxRef, "error", err)
		return
	}
	_ = resp.Body.Close()
	slog.Info("[MOCK CHAPA] callback called", "tx_ref", p.TxRef, "status_code", resp.StatusCode)
}
