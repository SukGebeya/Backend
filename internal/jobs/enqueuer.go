package jobs

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

// ErrNotReady tells a worker the work cannot complete yet (for example a payment
// still pending at the provider); the job is snoozed instead of failed.
var ErrNotReady = errors.New("not ready")

var ErrEnqueuerNotBound = errors.New("job enqueuer is not bound to a river client")

type Enqueuer interface {
	Enqueue(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) error
	EnqueueTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) error
	ScheduleCartExpiry(ctx context.Context, cartID uuid.UUID, expiresAt time.Time) error
}

// RiverEnqueuer is created before the River client so services and workers can
// be wired first; Bind attaches the client once it exists.
type RiverEnqueuer struct {
	mu     sync.RWMutex
	client *river.Client[pgx.Tx]
}

var _ Enqueuer = (*RiverEnqueuer)(nil)

func NewRiverEnqueuer() *RiverEnqueuer {
	return &RiverEnqueuer{}
}

func (e *RiverEnqueuer) Bind(client *river.Client[pgx.Tx]) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.client = client
}

func (e *RiverEnqueuer) get() (*river.Client[pgx.Tx], error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.client == nil {
		return nil, ErrEnqueuerNotBound
	}
	return e.client, nil
}

func (e *RiverEnqueuer) Enqueue(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) error {
	client, err := e.get()
	if err != nil {
		return err
	}
	_, err = client.Insert(ctx, args, opts)
	return err
}

func (e *RiverEnqueuer) EnqueueTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) error {
	client, err := e.get()
	if err != nil {
		return err
	}
	_, err = client.InsertTx(ctx, tx, args, opts)
	return err
}

func (e *RiverEnqueuer) ScheduleCartExpiry(ctx context.Context, cartID uuid.UUID, expiresAt time.Time) error {
	return e.Enqueue(ctx, MarkCartExpiredArgs{CartID: cartID}, &river.InsertOpts{ScheduledAt: expiresAt})
}

type NoopEnqueuer struct {
	mu   sync.Mutex
	Jobs []river.JobArgs
}

var _ Enqueuer = (*NoopEnqueuer)(nil)

func (n *NoopEnqueuer) Enqueue(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Jobs = append(n.Jobs, args)
	return nil
}

func (n *NoopEnqueuer) EnqueueTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) error {
	return n.Enqueue(ctx, args, opts)
}

func (n *NoopEnqueuer) ScheduleCartExpiry(ctx context.Context, cartID uuid.UUID, expiresAt time.Time) error {
	return n.Enqueue(ctx, MarkCartExpiredArgs{CartID: cartID}, nil)
}

func (n *NoopEnqueuer) Kinds() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	kinds := make([]string, len(n.Jobs))
	for i, j := range n.Jobs {
		kinds[i] = j.Kind()
	}
	return kinds
}
