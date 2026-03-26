package jobs

import (
	"github.com/google/uuid"
)

type ReleaseCartReservationArgs struct {
	CartID uuid.UUID `json:"cart_id"`
}

func (ReleaseCartReservationArgs) Kind() string {
	return "release_cart_reservation"
}

type MarkCartExpiredArgs struct {
	CartID uuid.UUID `json:"cart_id"`
}

func (MarkCartExpiredArgs) Kind() string {
	return "mark_cart_expired"
}

type ProcessPaymentWebhookArgs struct {
	TxRef  string `json:"tx_ref"`
	Status string `json:"status"`
}

func (ProcessPaymentWebhookArgs) Kind() string {
	return "process_payment_webhook"
}

type ProcessPayoutArgs struct {
	PayoutID uuid.UUID `json:"payout_id"`
}

func (ProcessPayoutArgs) Kind() string {
	return "process_payout"
}

type SendOrderNotificationArgs struct {
	OrderID uuid.UUID `json:"order_id"`
	Event   string    `json:"event"`
	Phone   string    `json:"phone"`
}

func (SendOrderNotificationArgs) Kind() string {
	return "send_order_notification"
}
