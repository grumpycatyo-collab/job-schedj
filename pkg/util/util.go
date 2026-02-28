package util

import (
	"context"

	"github.com/cenkalti/backoff/v5"
)

func RetryErr(ctx context.Context, op func() error, opts ...backoff.RetryOption) error {
	_, err := backoff.Retry[any](ctx, func() (any, error) {
		return nil, op()
	}, opts...)
	return err
}
