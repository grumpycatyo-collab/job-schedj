package scheduler

import (
	"context"
	"time"
)

type Store interface {
	SaveTask(ctx context.Context, task *Task) error
	ScheduleAt(ctx context.Context, taskID string, runAt time.Time) error
	PopDue(ctx context.Context, now time.Time, limit int) ([]Task, error)
}
