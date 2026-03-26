package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

const (
	NotificationEventPaymentConfirmed = "payment_confirmed"
	NotificationEventShipped          = "shipped"
	NotificationEventDelivered        = "delivered"
)

type SendOrderNotificationWorker struct {
	river.WorkerDefaults[SendOrderNotificationArgs]
}

func NewSendOrderNotificationWorker() *SendOrderNotificationWorker {
	return &SendOrderNotificationWorker{}
}

func (w *SendOrderNotificationWorker) Timeout(job *river.Job[SendOrderNotificationArgs]) time.Duration {
	return defaultJobTimeout
}

func (w *SendOrderNotificationWorker) Work(ctx context.Context, job *river.Job[SendOrderNotificationArgs]) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	orderID := job.Args.OrderID
	event := job.Args.Event
	phone := job.Args.Phone

	slog.Info("sending order notification",
		"order_id", orderID,
		"event", event,
		"phone", phone,
		"attempt", job.Attempt,
	)

	switch event {
	case NotificationEventPaymentConfirmed:
		w.sendPaymentConfirmation(ctx, orderID, phone)
	case NotificationEventShipped:
		w.sendShipmentNotification(ctx, orderID, phone)
	case NotificationEventDelivered:
		w.sendDeliveryConfirmation(ctx, orderID, phone)
	default:
		slog.Warn("unknown notification event", "event", event)
	}

	return nil
}

func (w *SendOrderNotificationWorker) sendPaymentConfirmation(ctx context.Context, orderID uuid.UUID, phone string) {
	slog.Info("[MOCK] SMS notification: Payment confirmed",
		"order_id", orderID,
		"phone", phone,
		"message", "Your order payment has been confirmed. Your order is now being processed.",
	)

	slog.Info("[MOCK] TODO: Implement SMS sending via Africa's Talking or similar SMS provider",
		"provider", "africas_talking",
		"phone", phone,
	)
}

func (w *SendOrderNotificationWorker) sendShipmentNotification(ctx context.Context, orderID uuid.UUID, phone string) {
	slog.Info("[MOCK] SMS notification: Order shipped",
		"order_id", orderID,
		"phone", phone,
		"message", "Your order has been shipped and is on its way.",
	)

	slog.Info("[MOCK] TODO: Implement SMS sending via Africa's Talking or similar SMS provider",
		"provider", "africas_talking",
		"phone", phone,
	)
}

func (w *SendOrderNotificationWorker) sendDeliveryConfirmation(ctx context.Context, orderID uuid.UUID, phone string) {
	slog.Info("[MOCK] SMS notification: Order delivered",
		"order_id", orderID,
		"phone", phone,
		"message", "Your order has been delivered. Thank you for shopping with us!",
	)

	slog.Info("[MOCK] TODO: Implement SMS sending via Africa's Talking or similar SMS provider",
		"provider", "africas_talking",
		"phone", phone,
	)
}
