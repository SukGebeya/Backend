package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type JobErrorHandler struct{}

func NewJobErrorHandler() *JobErrorHandler {
	return &JobErrorHandler{}
}

func (h *JobErrorHandler) HandleError(ctx context.Context, job *rivertype.JobRow, err error) *river.ErrorHandlerResult {
	slog.Error("job failed",
		"job_id", job.ID,
		"job_kind", job.Kind,
		"attempt", job.Attempt,
		"error", err,
	)
	return nil
}

func (h *JobErrorHandler) HandlePanic(ctx context.Context, job *rivertype.JobRow, panicValue any, trace string) *river.ErrorHandlerResult {
	slog.Error("job panicked",
		"job_id", job.ID,
		"job_kind", job.Kind,
		"attempt", job.Attempt,
		"panic_value", panicValue,
		"trace", trace,
	)
	return nil
}
