package util

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v5"
)

func RetryErr(ctx context.Context, op func() error, opts ...backoff.RetryOption) error {
	_, err := backoff.Retry[any](ctx, func() (any, error) {
		return nil, op()
	}, opts...)
	return err
}

func NextRunFromEvery(schedule string, now time.Time) (time.Time, error) {
	const prefix = "@every "
	// if schedule contains @every, strip it, and do time.ParseDuration()
	if !strings.HasPrefix(schedule, prefix) {
		return time.Time{}, errors.New("schedule must start with @")
	}

	raw := strings.TrimSpace(strings.TrimPrefix(schedule, prefix))
	if raw == "" {
		return time.Time{}, fmt.Errorf("missing duration in schedule %q", schedule)
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid duration %q: %v", schedule, err)
	}

	if d <= 0 {
		return time.Time{}, fmt.Errorf("invalid duration %q: must be > 0", schedule)
	}

	return now.Add(d), nil
}
