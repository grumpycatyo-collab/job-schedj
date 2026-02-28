package scheduler

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"example.com/m/config"
	"github.com/redis/go-redis/v9"
)

const (
	keySchedDue = "sched:due"
)

func keyJob(id string) string {
	return "job:" + id
}

type redisStore struct {
	mu     sync.Mutex
	client *redis.Client
}

func NewRedisStore(cfg config.RedisConfig) Store {
	r := &redisStore{
		client: redis.NewClient(&redis.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		}),
	}

	return r
}

func (r *redisStore) SaveTask(ctx context.Context, task *Task) error {
	stored := struct {
		ID       string        `json:"id"`
		Schedule string        `json:"schedule"`
		Timeout  time.Duration `json:"timeout"`
	}{
		ID:       task.ID,
		Schedule: task.Schedule,
		Timeout:  task.Timeout,
	}

	payload, err := json.Marshal(stored)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, keyJob(task.ID), payload, 0).Err()
}

func (r *redisStore) ScheduleAt(ctx context.Context, taskID string, runAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	score := float64(runAt.Unix())
	err := r.client.ZAdd(ctx, keySchedDue, redis.Z{
		Score:  score,
		Member: taskID,
	}).Err()

	if err != nil {
		return err
	}

	return nil
}

func (r *redisStore) PopDue(ctx context.Context, now time.Time, limit int) ([]Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ids, err := r.client.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     keySchedDue,
		Start:   "-inf",
		Stop:    strconv.FormatInt(now.Unix(), 10),
		ByScore: true,
		Offset:  0,
		Count:   int64(limit),
	}).Result()
	if err != nil {
		return nil, err
	}

	out := make([]Task, 0, len(ids))
	for _, id := range ids {
		removed, err := r.client.ZRem(ctx, keySchedDue, id).Result()
		if err != nil || removed == 0 {
			continue
		}
		raw, err := r.client.Get(ctx, keyJob(id)).Result()
		if err != nil {
			continue
		}

		var task Task
		if err := json.Unmarshal([]byte(raw), &task); err != nil {
			return nil, err
		}
		out = append(out, task)
	}

	return out, nil
}
