package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLock struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedis(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

func New(client *redis.Client, ttl time.Duration) *RedisLock {
	return &RedisLock{client: client, ttl: ttl}
}

// TryAcquire returns true if the lock was acquired.
func (l *RedisLock) TryAcquire(ctx context.Context, key, token string) (bool, error) {
	return l.client.SetNX(ctx, key, token, l.ttl).Result()
}

func (l *RedisLock) Release(ctx context.Context, key, token string) error {
	const script = `
if redis.call("get", KEYS[1]) == ARGV[1] then
  return redis.call("del", KEYS[1])
else
  return 0
end`
	_, err := l.client.Eval(ctx, script, []string{key}, token).Result()
	return err
}

func DriverKey(driverID string) string {
	return fmt.Sprintf("dispatch:driver:%s", driverID)
}

func LocationKey(driverID string) string {
	return fmt.Sprintf("driver:%s", driverID)
}
