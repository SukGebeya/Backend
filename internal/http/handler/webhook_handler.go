package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/SukGebeya/Backend/internal/chapa"
	apperrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/jobs"
	"github.com/SukGebeya/Backend/pkg/response"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type WebhookHandler struct {
	chapaClient *chapa.Client
	riverClient *river.Client[pgx.Tx]
}

func NewWebhookHandler(chapaClient *chapa.Client, riverClient *river.Client[pgx.Tx]) *WebhookHandler {
	return &WebhookHandler{
		chapaClient: chapaClient,
		riverClient: riverClient,
	}
}

type ChapaWebhookPayload struct {
	Event string `json:"event"`
	Data  struct {
		TxRef  string `json:"tx_ref"`
		Amount string `json:"amount"`
		Status string `json:"status"`
	} `json:"data"`
}

// HandleChapa godoc
// @Summary Handle Chapa webhook
// @Description Handle payment webhooks from Chapa payment gateway
// @Tags webhooks
// @Accept json
// @Produce json
// @Param Chapa-Signature header string true "Webhook signature"
// @Param payload body ChapaWebhookPayload true "Webhook payload"
// @Success 200 {object} map[string]string "Webhook received"
// @Failure 400 {object} response.ErrorResponse "Invalid payload"
// @Failure 401 {object} response.ErrorResponse "Invalid signature"
// @Router /webhooks/chapa [post]
func (h *WebhookHandler) HandleChapa(w http.ResponseWriter, r *http.Request) {
	signature := r.Header.Get("Chapa-Signature")
	if signature == "" {
		slog.Warn("HandleChapa: missing signature header")
		response.Error(w, apperrors.Unauthorized("missing webhook signature"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("HandleChapa: failed to read body", "error", err)
		response.Error(w, apperrors.BadRequest("failed to read request body"))
		return
	}
	defer r.Body.Close()

	if !h.chapaClient.VerifyWebhookSignature(body, signature) {
		slog.Warn("HandleChapa: invalid signature", "signature", signature)
		response.Error(w, apperrors.Unauthorized("invalid webhook signature"))
		return
	}

	var payload ChapaWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("HandleChapa: failed to parse payload", "error", err)
		response.Error(w, apperrors.BadRequest("invalid webhook payload"))
		return
	}

	slog.Info("HandleChapa: received webhook",
		"event", payload.Event,
		"tx_ref", payload.Data.TxRef,
		"status", payload.Data.Status,
	)

	if payload.Event != "charge.success" && payload.Event != "charge.failed" {
		response.JSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	txRef := payload.Data.TxRef
	if txRef == "" {
		slog.Error("HandleChapa: missing tx_ref")
		response.Error(w, apperrors.BadRequest("missing transaction reference"))
		return
	}

	_, err = h.riverClient.Insert(r.Context(), jobs.ProcessPaymentWebhookArgs{
		TxRef:  txRef,
		Status: payload.Data.Status,
	}, nil)
	if err != nil {
		slog.Error("HandleChapa: failed to enqueue webhook job",
			"error", err,
			"tx_ref", txRef,
		)
		response.Error(w, apperrors.Internal("failed to process webhook", err))
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "received"})
}
