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

	runCtx      context.Context
	runCancel   context.CancelFunc
	store       Store
	ready       chan *Task
	workerCount int
}

func NewScheduler(store Store) *Scheduler {
	logger := log.NewLogger()
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		logger:      logger,
		cron:        cron.New(),
		tasks:       make(map[string]cron.EntryID),
		runCtx:      ctx,
		runCancel:   cancel,
		store:       store,
		ready:       make(chan *Task, 256),
		workerCount: 4,
	}
}

func (s *Scheduler) Register(task Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.store.SaveTask(s.runCtx, &task); err != nil {
		s.logger.Error("failed to handle register", err)
		return err
	}

	runAt, err := util.NextRunFromEvery(task.Schedule, time.Now())
	if err != nil {
		return err
	}

	if err := s.store.ScheduleAt(s.runCtx, task.ID, runAt); err != nil {
		s.logger.Error("failed to handle schedule", err, task.ID)
		return err
	}

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

func (s *Scheduler) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case t := <-s.ready:
			if t == nil {
				continue
			}

			if err := s.safeExecute(ctx, *t); err != nil {
				retryAt := time.Now().Add(t.Timeout)
				if err := s.store.ScheduleAt(ctx, t.ID, retryAt); err != nil {
					s.logger.Error("failed to schedule retry on task", err, t.ID)
				}
				continue
			}

			nextRun, err := util.NextRunFromEvery(t.Schedule, time.Now())
			if err != nil {
				s.logger.Error("failed to schedule next run", err)
				continue
			}

			if err := s.store.ScheduleAt(ctx, t.ID, nextRun); err != nil {
				s.logger.Error("failed to schedule next run", err)
			}
		}
	}
}

func (s *Scheduler) startWorkers(ctx context.Context) {
	for i := 0; i < s.workerCount; i++ {
		go s.worker(ctx)
	}
}

func (s *Scheduler) dispatchLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			due, err := s.store.PopDue(ctx, time.Now(), 100)
			if err != nil {
				s.logger.Error("failed to pop due time", err)
			}

			for _, task := range due {
				t := &task
				select {
				case <-ctx.Done():
					return
				case s.ready <- t:
				}
			}

		}
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.startWorkers(ctx)

	go s.dispatchLoop(ctx)

	<-ctx.Done()
	s.runCancel()
	close(s.ready)
	return nil
}
