package domain

import (
	"time"

	"github.com/google/uuid"
)

type WebhookEventStatus string

const (
	WebhookEventStatusReceived  WebhookEventStatus = "received"
	WebhookEventStatusProcessed WebhookEventStatus = "processed"
	WebhookEventStatusIgnored   WebhookEventStatus = "ignored"
	WebhookEventStatusFailed    WebhookEventStatus = "failed"
)

type WebhookEvent struct {
	ID          uuid.UUID          `json:"id"`
	Provider    string             `json:"provider"`
	EventType   string             `json:"event_type"`
	TxRef       *string            `json:"tx_ref,omitempty"`
	PayloadHash string             `json:"payload_hash"`
	Payload     []byte             `json:"payload"`
	Status      WebhookEventStatus `json:"status"`
	Error       *string            `json:"error,omitempty"`
	ReceivedAt  time.Time          `json:"received_at"`
	ProcessedAt *time.Time         `json:"processed_at,omitempty"`
}
