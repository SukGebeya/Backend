package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/SukGebeya/Backend/internal/chapa"
	"github.com/SukGebeya/Backend/internal/domain"
	apperrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/jobs"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/pkg/response"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const maxWebhookBodyBytes = 1 << 20

type WebhookHandler struct {
	chapaClient   chapa.ChapaClient
	webhookEvents repository.WebhookEventRepository
	txManager     repository.TransactionManager
	enqueuer      jobs.Enqueuer
}

func NewWebhookHandler(chapaClient chapa.ChapaClient, webhookEvents repository.WebhookEventRepository, txManager repository.TransactionManager, enqueuer jobs.Enqueuer) *WebhookHandler {
	return &WebhookHandler{
		chapaClient:   chapaClient,
		webhookEvents: webhookEvents,
		txManager:     txManager,
		enqueuer:      enqueuer,
	}
}

type WebhookAckResponse struct {
	Status string `json:"status" example:"received"`
}

// HandleChapa godoc
// @Summary Handle Chapa webhook
// @Description Receives Chapa payment events. The signature is checked, the event is recorded once, and the payment is re-verified with Chapa asynchronously before any order changes.
// @Tags webhooks
// @Accept json
// @Produce json
// @Param x-chapa-signature header string false "HMAC-SHA256 of the payload"
// @Param Chapa-Signature header string false "HMAC-SHA256 of the webhook secret"
// @Param payload body chapa.WebhookPayload true "Webhook payload"
// @Success 200 {object} WebhookAckResponse
// @Failure 400 {object} response.ErrorResponse "Invalid payload"
// @Failure 401 {object} response.ErrorResponse "Invalid signature"
// @Router /webhooks/chapa [post]
func (h *WebhookHandler) HandleChapa(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodyBytes))
	if err != nil {
		response.Error(w, apperrors.BadRequest("failed to read request body"))
		return
	}

	if !h.chapaClient.VerifyWebhookSignature(body, r.Header.Get("x-chapa-signature"), r.Header.Get("Chapa-Signature")) {
		slog.Warn("chapa webhook rejected: invalid signature", "remote_ip", r.RemoteAddr)
		response.Error(w, apperrors.Unauthorized("invalid webhook signature"))
		return
	}

	var payload chapa.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		response.Error(w, apperrors.BadRequest("invalid webhook payload"))
		return
	}

	txRef := payload.Reference()
	slog.Info("chapa webhook received", "event", payload.Event, "tx_ref", txRef, "status", payload.PaymentStatus())

	hash := sha256.Sum256(body)
	event := &domain.WebhookEvent{
		ID:          uuid.New(),
		Provider:    "chapa",
		EventType:   payload.Event,
		PayloadHash: hex.EncodeToString(hash[:]),
		Payload:     body,
		Status:      domain.WebhookEventStatusReceived,
	}
	if txRef != "" {
		event.TxRef = &txRef
	}

	kind := classifyChapaEvent(payload.Event, txRef)
	if kind == eventReversal && !h.chapaClient.VerifyPayloadSignature(body, r.Header.Get("x-chapa-signature")) {
		slog.Warn("chapa reversal webhook lacks a payload signature; recorded for manual review", "tx_ref", txRef)
		kind = eventIgnored
	}
	if kind == eventIgnored {
		event.Status = domain.WebhookEventStatusIgnored
	}

	err = h.txManager.Run(r.Context(), func(ctx context.Context, tx pgx.Tx) error {
		recorded, created, err := h.webhookEvents.WithTx(tx).Create(ctx, event)
		if err != nil {
			return err
		}
		if !created {
			slog.Info("chapa webhook duplicate ignored", "tx_ref", txRef)
			return nil
		}
		switch kind {
		case eventPayment:
			return h.enqueuer.EnqueueTx(ctx, tx, jobs.ProcessPaymentWebhookArgs{TxRef: txRef, WebhookEventID: &recorded.ID}, nil)
		case eventReversal:
			return h.enqueuer.EnqueueTx(ctx, tx, jobs.ProcessPaymentReversalArgs{TxRef: txRef, WebhookEventID: &recorded.ID}, nil)
		default:
			return nil
		}
	})
	if err != nil {
		slog.Error("failed to record chapa webhook", "tx_ref", txRef, "error", err)
		response.Error(w, apperrors.Internal("failed to process webhook", err))
		return
	}

	response.JSON(w, http.StatusOK, WebhookAckResponse{Status: "received"})
}

type chapaEventKind int

const (
	eventIgnored chapaEventKind = iota
	eventPayment
	eventReversal
)

// classifyChapaEvent maps Chapa event names to how they are handled. Payment
// events are always re-verified through the API; reversals move money back out of
// the merchant's hands and therefore also require the payload HMAC.
func classifyChapaEvent(event, txRef string) chapaEventKind {
	if txRef == "" {
		return eventIgnored
	}
	switch event {
	case "", "charge.success", "charge.failed", "charge.cancelled", "charge.failed/cancelled":
		return eventPayment
	case "charge.reversed", "charge.refunded":
		return eventReversal
	default:
		return eventIgnored
	}
}

// ChapaCallback godoc
// @Summary Chapa payment callback
// @Description Chapa calls this URL (CHAPA_CALLBACK_URL) after checkout. The query is only a hint; the payment is verified with Chapa before the order changes.
// @Tags webhooks
// @Produce json
// @Param trx_ref query string false "Transaction reference"
// @Param tx_ref query string false "Transaction reference"
// @Success 200 {object} WebhookAckResponse
// @Failure 400 {object} response.ErrorResponse
// @Router /payments/chapa/callback [get]
func (h *WebhookHandler) ChapaCallback(w http.ResponseWriter, r *http.Request) {
	txRef := r.URL.Query().Get("trx_ref")
	if txRef == "" {
		txRef = r.URL.Query().Get("tx_ref")
	}
	if txRef == "" {
		response.Error(w, apperrors.BadRequest("missing transaction reference"))
		return
	}

	if err := h.enqueuer.Enqueue(r.Context(), jobs.ProcessPaymentWebhookArgs{TxRef: txRef}, nil); err != nil {
		response.Error(w, apperrors.Internal("failed to queue payment verification", err))
		return
	}

	response.JSON(w, http.StatusOK, WebhookAckResponse{Status: "received"})
}
