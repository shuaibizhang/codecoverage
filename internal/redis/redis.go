package redis

import (
	"context"

	"github.com/gomodule/redigo/redis"
)

// RedisIF defines the interface for redis operations used in the project.
type RedisIF interface {
	Del(ctx context.Context, key string) (int, error)
	Eval(ctx context.Context, script string, keyCount int, keysAndArgs []interface{}) (interface{}, error)
	EvalShaWithHash(ctx context.Context, hash string, keyCount int, keysAndArgs []interface{}) (interface{}, error)
}

// Int is a helper that converts a redis reply to an integer.
func Int(reply interface{}, err error) (int, error) {
	return redis.Int(reply, err)
}

// String is a helper that converts a redis reply to a string.
func String(reply interface{}, err error) (string, error) {
	return redis.String(reply, err)
}

// Bool is a helper that converts a redis reply to a boolean.
func Bool(reply interface{}, err error) (bool, error) {
	return redis.Bool(reply, err)
}
