package reportlock

import (
	"context"

	"github.com/shuaibizhang/codecoverage/internal/dislock"
	libredis "github.com/shuaibizhang/codecoverage/internal/redis"
)

type ReportLock interface {
	Lock(ctx context.Context) error
	Unlock(ctx context.Context) error
	CanWrite(ctx context.Context) bool
}

type reportLock struct {
	rwLock dislock.RWLock
}

func NewReportLock(logger dislock.Logger, redisClient libredis.RedisIF, lockKey string) ReportLock {
	return &reportLock{
		rwLock: dislock.NewRWLock(logger, redisClient, lockKey),
	}
}

func (l *reportLock) Lock(ctx context.Context) error {
	// 报告锁默认使用写锁
	return l.rwLock.Lock(ctx, true)
}

func (l *reportLock) Unlock(ctx context.Context) error {
	return l.rwLock.Unlock(ctx)
}

func (l *reportLock) CanWrite(ctx context.Context) bool {
	return l.rwLock.CanWrite()
}
