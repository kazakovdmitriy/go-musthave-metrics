package retry

import (
	"context"
	"time"
)

type Operation func() error
type IsRetryableError func(error) bool

type RetryConfig struct {
	MaxRetries    int
	Delays        []time.Duration
	IsRetryableFn IsRetryableError
}

func Do(ctx context.Context, cfg RetryConfig, op Operation) error {
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}

	if cfg.Delays == nil {
		cfg.Delays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}
	}

	if cfg.IsRetryableFn == nil {
		cfg.IsRetryableFn = func(error) bool { return false }
	}

	totalAttempts := cfg.MaxRetries + 1

	var lastErr error

	for i := 0; i < totalAttempts; i++ {
		err := op()
		if err == nil {
			return nil
		}

		if !cfg.IsRetryableFn(err) {
			return err
		}

		lastErr = err

		if i < len(cfg.Delays) {
			select {
			case <-time.After(cfg.Delays[i]):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	if lastErr != nil {
		return lastErr
	}
	return ctx.Err()
}
