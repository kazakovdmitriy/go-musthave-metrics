package ping

import "context"

type HealthChecker interface {
	Ping(ctx context.Context) error
}
