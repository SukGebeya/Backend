package jobs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type JobScheduler interface {
	ScheduleCartExpiry(ctx context.Context, cartID uuid.UUID, expiresAt time.Time) error
}

type RiverJobScheduler struct {
	client *river.Client[pgx.Tx]
}

func NewRiverJobScheduler(client *river.Client[pgx.Tx]) *RiverJobScheduler {
	return &RiverJobScheduler{client: client}
}

func (s *RiverJobScheduler) ScheduleCartExpiry(ctx context.Context, cartID uuid.UUID, expiresAt time.Time) error {
	_, err := s.client.Insert(ctx, MarkCartExpiredArgs{CartID: cartID}, &river.InsertOpts{
		ScheduledAt: expiresAt,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true,
		},
	})
	return err
}
