package env

import "context"

type Environment interface {
	Init(ctx context.Context) error
	IsReady(ctx context.Context) bool
	Reload(ctx context.Context) error
}
