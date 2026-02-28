package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"example.com/m/pkg/log"
	"example.com/m/pkg/util"
	"github.com/cenkalti/backoff/v5"
	"github.com/robfig/cron/v3"
)

type Task struct {
	ID       string
	Schedule string
	Handler  func(ctx context.Context) error
	Timeout  time.Duration
}

type Scheduler struct {
	logger log.Logger
	cron   *cron.Cron
	tasks  map[string]cron.EntryID
	mu     sync.RWMutex

	runCtx    context.Context
	runCancel context.CancelFunc
}

func NewScheduler() *Scheduler {
	logger := log.NewLogger()
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		logger:    logger,
		cron:      cron.New(),
		tasks:     make(map[string]cron.EntryID),
		runCtx:    ctx,
		runCancel: cancel,
	}
}

func (s *Scheduler) Register(task Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := s.cron.AddFunc(task.Schedule, func() {
		if err := s.safeExecute(s.runCtx, task); err != nil {
			s.logger.Error("failed to handle register", err)
		}
	})
	if err != nil {
		return err
	}

	s.tasks[task.ID] = id
	return nil
}

func (s *Scheduler) safeExecute(ctx context.Context, task Task) error {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("task panicked: %v", r)
			s.logger.Error("task panicked", err)
		}
	}()

	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 10 * time.Minute // optional cap

	if err := util.RetryErr(ctx, func() error {
		return task.Handler(ctx)
	}, backoff.WithBackOff(bo)); err != nil {
		s.logger.Info("Task %s failed after retries: %v", task.ID, err)
		return err
	}

	return nil
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.cron.Start()

	<-ctx.Done()
	s.runCancel()

	stopCtx := s.cron.Stop()

	select {
	case <-stopCtx.Done():
		s.logger.Info("All tasks completed")
	case <-time.After(30 * time.Second):
		s.logger.Info("Shutdown timeout exceeded")
	}

	return nil
}
