package dislock

// 分布式锁
import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"sync"

	redigo "github.com/gomodule/redigo/redis"
	libredis "github.com/shuaibizhang/codecoverage/internal/redis"
)

// Logger 接口，用于打印日志
type Logger interface {
	Infof(ctx context.Context, tag string, format string, args ...interface{})
}

const (
	DLTagDistributeLock = "_distribute_lock"
)

// RWLock 分布式读写锁接口
type RWLock interface {
	Lock(ctx context.Context, write bool) error
	Unlock(ctx context.Context) error
	CanWrite() bool
	Clean(ctx context.Context) (int, error)
}

type lockOp uint8

const (
	opLockRead    lockOp = 1
	opUnlockRead  lockOp = 2
	opLockWrite   lockOp = 3
	opUnlockWrite lockOp = 4
)

//go:embed lock.lua
var luaLockScript string
var lockScript = redigo.NewScript(1, luaLockScript)

var (
	ErrLockFail   = errors.New("lock fail")
	ErrUnlockFail = errors.New("unlock fail")
)

type distributeRWLock struct {
	logger      Logger
	redisClient libredis.RedisIF
	mu          sync.Mutex
	lockKey     string
	lastOp      lockOp
}

// NewRWLock 创建一个通用的分布式读写锁
func NewRWLock(logger Logger, redisClient libredis.RedisIF, lockKey string) RWLock {
	return &distributeRWLock{
		logger:      logger,
		redisClient: redisClient,
		lockKey:     lockKey,
	}
}

// Lock 加锁
func (l *distributeRWLock) Lock(ctx context.Context, write bool) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.lastOp != 0 {
		return fmt.Errorf("last op is %d", l.lastOp)
	}

	l.logger.Infof(ctx, DLTagDistributeLock, "key=%s||write=%t||_msg=lock", l.lockKey, write)

	if write {
		return l.lockWrite(ctx)
	}
	return l.lockRead(ctx)
}

// Unlock 解锁
func (l *distributeRWLock) Unlock(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 使用 background context 确保即使外部 ctx cancel 也能完成解锁
	newCtx := context.Background()

	if l.lastOp == opLockWrite {
		l.logger.Infof(ctx, DLTagDistributeLock, "key=%s||_msg=unlock write", l.lockKey)
		return l.unlockWrite(newCtx)
	}
	if l.lastOp == opLockRead {
		l.logger.Infof(ctx, DLTagDistributeLock, "key=%s||_msg=unlock read", l.lockKey)
		return l.unlockRead(newCtx)
	}
	return fmt.Errorf("last op is %d", l.lastOp)
}

// CanWrite 检查当前是否持有写锁
func (l *distributeRWLock) CanWrite() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.lastOp == opLockWrite
}

// Clean 仅用于测试: 删除对应的 redis key
func (l *distributeRWLock) Clean(ctx context.Context) (int, error) {
	return l.redisClient.Del(ctx, l.lockKey)
}

func (l *distributeRWLock) lockWrite(ctx context.Context) error {
	reply, err := l.eval(ctx, opLockWrite)
	res, err := libredis.Int(reply, err)
	if err != nil {
		return err
	}
	if res <= 0 {
		return ErrLockFail
	}
	l.lastOp = opLockWrite
	return nil
}

func (l *distributeRWLock) unlockWrite(ctx context.Context) error {
	reply, err := l.eval(ctx, opUnlockWrite)
	res, err := libredis.Int(reply, err)
	if err != nil {
		return err
	}
	if res <= 0 {
		return ErrUnlockFail
	}
	l.lastOp = 0
	return nil
}

func (l *distributeRWLock) lockRead(ctx context.Context) error {
	reply, err := l.eval(ctx, opLockRead)
	res, err := libredis.Int(reply, err)
	if err != nil {
		return err
	}
	if res <= 0 {
		return ErrLockFail
	}
	l.lastOp = opLockRead
	return nil
}

func (l *distributeRWLock) unlockRead(ctx context.Context) error {
	reply, err := l.eval(ctx, opUnlockRead)
	res, err := libredis.Int(reply, err)
	if err != nil {
		return err
	}
	if res <= 0 {
		return ErrUnlockFail
	}
	l.lastOp = 0
	return nil
}

func (l *distributeRWLock) eval(ctx context.Context, op lockOp) (reply interface{}, err error) {
	keyAndArgs := []interface{}{l.lockKey, op}
	reply, err = l.redisClient.EvalShaWithHash(ctx, lockScript.Hash(), 1, keyAndArgs)
	if err != nil && strings.HasPrefix(err.Error(), "NOSCRIPT") {
		return l.redisClient.Eval(ctx, luaLockScript, 1, keyAndArgs)
	}
	return reply, err
}
