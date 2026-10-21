package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/pkg/sms"
	"github.com/riverqueue/river"
)

const (
	NotificationEventPaymentConfirmed = "payment_confirmed"
	NotificationEventShipped          = "shipped"
	NotificationEventDelivered        = "delivered"
	NotificationEventCancelled        = "cancelled"
	NotificationEventRefunded         = "refunded"
)

var notificationMessages = map[string]string{
	NotificationEventPaymentConfirmed: "ገበያ፦ ክፍያዎ ተረጋግጧል፤ ነጋዴው ትዕዛዝዎን እያዘጋጀ ነው። Gebeya: payment confirmed, the merchant is preparing order %s.",
	NotificationEventShipped:          "ገበያ፦ ትዕዛዝዎ ተልኳል። ሲደርስዎ በመተግበሪያው ያረጋግጡ። Gebeya: order %s has shipped. Confirm in the app when it arrives.",
	NotificationEventDelivered:        "ገበያ፦ ትዕዛዝዎ ደርሷል። እናመሰግናለን! Gebeya: order %s delivered. Thank you!",
	NotificationEventCancelled:        "ገበያ፦ ትዕዛዝዎ ተሰርዟል። Gebeya: order %s was cancelled. Any payment will be refunded.",
	NotificationEventRefunded:         "ገበያ፦ ክፍያዎ ተመልሷል። Gebeya: the payment for order %s has been refunded.",
}

type SendOrderNotificationWorker struct {
	river.WorkerDefaults[SendOrderNotificationArgs]
	orderRepo repository.OrderRepository
	userRepo  repository.UserRepository
	sms       sms.Sender
}

func (w *SendOrderNotificationWorker) Timeout(*river.Job[SendOrderNotificationArgs]) time.Duration {
	return defaultJobTimeout
}

// Work texts the buyer about their order in Amharic and English.
func (w *SendOrderNotificationWorker) Work(ctx context.Context, job *river.Job[SendOrderNotificationArgs]) error {
	message, ok := notificationMessages[job.Args.Event]
	if !ok {
		slog.Warn("unknown notification event", "event", job.Args.Event)
		return nil
	}

	order, err := w.orderRepo.GetByID(ctx, job.Args.OrderID)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	user, err := w.userRepo.GetByID(ctx, order.BuyerID)
	if err != nil {
		if domainerrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if user.Phone == nil || *user.Phone == "" {
		slog.Info("skipping notification: buyer has no phone", "order_id", order.ID)
		return nil
	}

	if w.sms == nil {
		slog.Info("order notification (no SMS sender configured)", "order_id", order.ID, "event", job.Args.Event)
		return nil
	}
	return w.sms.Send(ctx, *user.Phone, fmt.Sprintf(message, order.ID.String()[:8]))
}
