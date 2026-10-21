package jobs

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/pkg/sms"
	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

const (
	defaultJobTimeout     = 60 * time.Second
	paymentPendingSnooze  = time.Minute
	transferPendingSnooze = 2 * time.Minute
	maxPendingAge         = 7 * 24 * time.Hour
)

type CartExpirer interface {
	ExpireCart(ctx context.Context, cartID uuid.UUID) error
}

type PaymentProcessor interface {
	ProcessPayment(ctx context.Context, txRef string) error
	ProcessReversal(ctx context.Context, txRef string) error
	ExpireUnpaidOrder(ctx context.Context, orderID uuid.UUID) error
	RefundOrder(ctx context.Context, orderID uuid.UUID) error
}

type OrderSettler interface {
	SettleOrder(ctx context.Context, orderID uuid.UUID) error
	AutoConfirmDelivery(ctx context.Context, orderID uuid.UUID) error
}

type Reconciler interface {
	Reconcile(ctx context.Context) error
}

type PayoutProcessor interface {
	ProcessPayout(ctx context.Context, payoutID uuid.UUID) error
}

type WorkerDeps struct {
	Carts         CartExpirer
	Payments      PaymentProcessor
	Orders        OrderSettler
	Payouts       PayoutProcessor
	Reconciler    Reconciler
	SMS           sms.Sender
	WebhookEvents repository.WebhookEventRepository
	OrderRepo     repository.OrderRepository
	UserRepo      repository.UserRepository
}

func AddWorkers(workers *river.Workers, deps WorkerDeps) {
	river.AddWorker(workers, &MarkCartExpiredWorker{carts: deps.Carts})
	river.AddWorker(workers, &ProcessPaymentWebhookWorker{payments: deps.Payments, webhookEvents: deps.WebhookEvents})
	river.AddWorker(workers, &ExpireUnpaidOrderWorker{payments: deps.Payments})
	river.AddWorker(workers, &RefundOrderWorker{payments: deps.Payments})
	river.AddWorker(workers, &SettleOrderWorker{orders: deps.Orders})
	river.AddWorker(workers, &AutoConfirmDeliveryWorker{orders: deps.Orders})
	river.AddWorker(workers, &ProcessPaymentReversalWorker{payments: deps.Payments, webhookEvents: deps.WebhookEvents})
	river.AddWorker(workers, &ReconcileLedgerWorker{reconciler: deps.Reconciler})
	river.AddWorker(workers, &ProcessPayoutWorker{payouts: deps.Payouts})
	river.AddWorker(workers, &SendOrderNotificationWorker{orderRepo: deps.OrderRepo, userRepo: deps.UserRepo, sms: deps.SMS})
}

// outcome converts service errors into River semantics: not-ready work is
// snoozed, permanent client errors are cancelled, everything else retries.
func outcome(job *rivertype.JobRow, err error, snooze time.Duration) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, blnk.ErrInflightBusy) {
		return river.JobSnooze(10 * time.Second)
	}
	if errors.Is(err, ErrNotReady) {
		if time.Since(job.CreatedAt) > maxPendingAge {
			return river.JobCancel(err)
		}
		return river.JobSnooze(snooze)
	}
	if appErr, ok := domainerrors.IsAppError(err); ok && appErr.HTTPStatus >= 400 && appErr.HTTPStatus < 500 {
		slog.Warn("job cancelled: permanent error", "job_id", job.ID, "kind", job.Kind, "error", err)
		return river.JobCancel(err)
	}
	return err
}

type MarkCartExpiredWorker struct {
	river.WorkerDefaults[MarkCartExpiredArgs]
	carts CartExpirer
}

func (w *MarkCartExpiredWorker) Timeout(*river.Job[MarkCartExpiredArgs]) time.Duration {
	return defaultJobTimeout
}

func (w *MarkCartExpiredWorker) Work(ctx context.Context, job *river.Job[MarkCartExpiredArgs]) error {
	return outcome(job.JobRow, w.carts.ExpireCart(ctx, job.Args.CartID), time.Hour)
}

type ProcessPaymentWebhookWorker struct {
	river.WorkerDefaults[ProcessPaymentWebhookArgs]
	payments      PaymentProcessor
	webhookEvents repository.WebhookEventRepository
}

func (w *ProcessPaymentWebhookWorker) Timeout(*river.Job[ProcessPaymentWebhookArgs]) time.Duration {
	return defaultJobTimeout
}

func (w *ProcessPaymentWebhookWorker) Work(ctx context.Context, job *river.Job[ProcessPaymentWebhookArgs]) error {
	slog.Info("processing payment", "tx_ref", job.Args.TxRef, "attempt", job.Attempt)

	err := w.payments.ProcessPayment(ctx, job.Args.TxRef)
	result := outcome(job.JobRow, err, paymentPendingSnooze)

	if job.Args.WebhookEventID != nil && w.webhookEvents != nil && !errors.Is(err, ErrNotReady) {
		status := domain.WebhookEventStatusProcessed
		var msg *string
		if err != nil {
			status = domain.WebhookEventStatusFailed
			text := err.Error()
			msg = &text
		}
		if updateErr := w.webhookEvents.UpdateStatus(ctx, *job.Args.WebhookEventID, status, msg); updateErr != nil {
			slog.Error("failed to update webhook event status", "webhook_event_id", *job.Args.WebhookEventID, "error", updateErr)
		}
	}

	return result
}

type ProcessPaymentReversalWorker struct {
	river.WorkerDefaults[ProcessPaymentReversalArgs]
	payments      PaymentProcessor
	webhookEvents repository.WebhookEventRepository
}

func (w *ProcessPaymentReversalWorker) Timeout(*river.Job[ProcessPaymentReversalArgs]) time.Duration {
	return defaultJobTimeout
}

func (w *ProcessPaymentReversalWorker) Work(ctx context.Context, job *river.Job[ProcessPaymentReversalArgs]) error {
	err := w.payments.ProcessReversal(ctx, job.Args.TxRef)
	if job.Args.WebhookEventID != nil && w.webhookEvents != nil {
		status := domain.WebhookEventStatusProcessed
		var msg *string
		if err != nil {
			status = domain.WebhookEventStatusFailed
			text := err.Error()
			msg = &text
		}
		if updateErr := w.webhookEvents.UpdateStatus(ctx, *job.Args.WebhookEventID, status, msg); updateErr != nil {
			slog.Error("failed to update webhook event status", "webhook_event_id", *job.Args.WebhookEventID, "error", updateErr)
		}
	}
	return outcome(job.JobRow, err, time.Minute)
}

type AutoConfirmDeliveryWorker struct {
	river.WorkerDefaults[AutoConfirmDeliveryArgs]
	orders OrderSettler
}

func (w *AutoConfirmDeliveryWorker) Timeout(*river.Job[AutoConfirmDeliveryArgs]) time.Duration {
	return defaultJobTimeout
}

func (w *AutoConfirmDeliveryWorker) Work(ctx context.Context, job *river.Job[AutoConfirmDeliveryArgs]) error {
	return outcome(job.JobRow, w.orders.AutoConfirmDelivery(ctx, job.Args.OrderID), time.Hour)
}

type ReconcileLedgerWorker struct {
	river.WorkerDefaults[ReconcileLedgerArgs]
	reconciler Reconciler
}

func (w *ReconcileLedgerWorker) Timeout(*river.Job[ReconcileLedgerArgs]) time.Duration {
	return 10 * time.Minute
}

func (w *ReconcileLedgerWorker) Work(ctx context.Context, job *river.Job[ReconcileLedgerArgs]) error {
	if w.reconciler == nil {
		return nil
	}
	return w.reconciler.Reconcile(ctx)
}

// DailyAt is a River periodic schedule that fires once a day at a wall-clock time
// in a given location (e.g. 03:00 Africa/Addis_Ababa).
type DailyAt struct {
	Hour     int
	Location *time.Location
}

func (d DailyAt) Next(current time.Time) time.Time {
	local := current.In(d.Location)
	next := time.Date(local.Year(), local.Month(), local.Day(), d.Hour, 0, 0, 0, d.Location)
	if !next.After(local) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

type ExpireUnpaidOrderWorker struct {
	river.WorkerDefaults[ExpireUnpaidOrderArgs]
	payments PaymentProcessor
}

func (w *ExpireUnpaidOrderWorker) Timeout(*river.Job[ExpireUnpaidOrderArgs]) time.Duration {
	return defaultJobTimeout
}

func (w *ExpireUnpaidOrderWorker) Work(ctx context.Context, job *river.Job[ExpireUnpaidOrderArgs]) error {
	return outcome(job.JobRow, w.payments.ExpireUnpaidOrder(ctx, job.Args.OrderID), paymentPendingSnooze)
}

type RefundOrderWorker struct {
	river.WorkerDefaults[RefundOrderArgs]
	payments PaymentProcessor
}

func (w *RefundOrderWorker) Timeout(*river.Job[RefundOrderArgs]) time.Duration {
	return defaultJobTimeout
}

func (w *RefundOrderWorker) Work(ctx context.Context, job *river.Job[RefundOrderArgs]) error {
	return outcome(job.JobRow, w.payments.RefundOrder(ctx, job.Args.OrderID), transferPendingSnooze)
}

type SettleOrderWorker struct {
	river.WorkerDefaults[SettleOrderArgs]
	orders OrderSettler
}

func (w *SettleOrderWorker) Timeout(*river.Job[SettleOrderArgs]) time.Duration {
	return defaultJobTimeout
}

func (w *SettleOrderWorker) Work(ctx context.Context, job *river.Job[SettleOrderArgs]) error {
	return outcome(job.JobRow, w.orders.SettleOrder(ctx, job.Args.OrderID), time.Minute)
}

type ProcessPayoutWorker struct {
	river.WorkerDefaults[ProcessPayoutArgs]
	payouts PayoutProcessor
}

func (w *ProcessPayoutWorker) Timeout(*river.Job[ProcessPayoutArgs]) time.Duration {
	return defaultJobTimeout
}

func (w *ProcessPayoutWorker) Work(ctx context.Context, job *river.Job[ProcessPayoutArgs]) error {
	return outcome(job.JobRow, w.payouts.ProcessPayout(ctx, job.Args.PayoutID), transferPendingSnooze)
}
