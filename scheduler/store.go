package scheduler

import (
	"container/heap"
	"context"
	"sync"
	"time"
)

type Store interface {
	SaveTask(ctx context.Context, task *Task) error
	ScheduleAt(ctx context.Context, taskID string, runAt time.Time) error
	PopDue(ctx context.Context, now time.Time, limit int) ([]*Task, error)
}

type dueItem struct {
	taskID string
	runAt  time.Time
	index  int // heap index
}

type dueHeap []*dueItem

func (h dueHeap) Len() int           { return len(h) }
func (h dueHeap) Less(i, j int) bool { return h[i].runAt.Before(h[j].runAt) }
func (h dueHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i]; h[i].index = i; h[j].index = j }
func (h *dueHeap) Push(x any)        { *h = append(*h, x.(*dueItem)) }
func (h *dueHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

type memStore struct {
	mu    sync.Mutex
	tasks map[string]Task
	due   dueHeap
}

func (m *memStore) SaveTask(_ context.Context, t Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[t.ID] = t
	return nil
}

func (m *memStore) ScheduleAt(_ context.Context, taskID string, runAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	heap.Push(&m.due, &dueItem{taskID: taskID, runAt: runAt})
	return nil
}

func (m *memStore) PopDue(_ context.Context, now time.Time, limit int) ([]Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Task, 0, limit)
	for len(out) < limit && m.due.Len() > 0 {
		top := m.due[0]
		if top.runAt.After(now) {
			break
		}
		it := heap.Pop(&m.due).(*dueItem)
		t, ok := m.tasks[it.taskID]
		if ok {
			out = append(out, t)
		}
	}
	return out, nil
}
