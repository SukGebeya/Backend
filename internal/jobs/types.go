package jobs

import (
	"github.com/google/uuid"
	"github.com/riverqueue/river"
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

func (MarkCartExpiredArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

type ProcessPaymentWebhookArgs struct {
	TxRef          string     `json:"tx_ref" river:"unique"`
	WebhookEventID *uuid.UUID `json:"webhook_event_id,omitempty"`
}

func (ProcessPaymentWebhookArgs) Kind() string {
	return "process_payment_webhook"
}

func (ProcessPaymentWebhookArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

type ExpireUnpaidOrderArgs struct {
	OrderID uuid.UUID `json:"order_id"`
}

func (ExpireUnpaidOrderArgs) Kind() string {
	return "expire_unpaid_order"
}

func (ExpireUnpaidOrderArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

type RefundOrderArgs struct {
	OrderID uuid.UUID `json:"order_id"`
}

func (RefundOrderArgs) Kind() string {
	return "refund_order"
}

func (RefundOrderArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}, MaxAttempts: 10}
}

type SettleOrderArgs struct {
	OrderID uuid.UUID `json:"order_id"`
}

func (SettleOrderArgs) Kind() string {
	return "settle_order"
}

func (SettleOrderArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}, MaxAttempts: 25}
}

type ProcessPayoutArgs struct {
	PayoutID uuid.UUID `json:"payout_id"`
}

func (ProcessPayoutArgs) Kind() string {
	return "process_payout"
}

func (ProcessPayoutArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}, MaxAttempts: 10}
}

type SendOrderNotificationArgs struct {
	OrderID uuid.UUID `json:"order_id"`
	Event   string    `json:"event"`
}

func (SendOrderNotificationArgs) Kind() string {
	return "send_order_notification"
}

type AutoConfirmDeliveryArgs struct {
	OrderID uuid.UUID `json:"order_id"`
}

func (AutoConfirmDeliveryArgs) Kind() string {
	return "auto_confirm_delivery"
}

func (AutoConfirmDeliveryArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

type ProcessPaymentReversalArgs struct {
	TxRef          string     `json:"tx_ref" river:"unique"`
	WebhookEventID *uuid.UUID `json:"webhook_event_id,omitempty"`
}

func (ProcessPaymentReversalArgs) Kind() string {
	return "process_payment_reversal"
}

func (ProcessPaymentReversalArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}, MaxAttempts: 15}
}

type ReconcileLedgerArgs struct{}

func (ReconcileLedgerArgs) Kind() string {
	return "reconcile_ledger"
}
