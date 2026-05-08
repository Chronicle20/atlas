package lock

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// TenantLock wraps Redis SETNX-with-owner semantics for the per-tenant
// extraction lock. The Lua compare-and-delete on Release prevents a stale
// holder (whose TTL just expired) from accidentally releasing a lock that has
// since been re-acquired by a different job.
type TenantLock struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewTenantLock(client *goredis.Client, ttl time.Duration) *TenantLock {
	return &TenantLock{client: client, ttl: ttl}
}

func (t *TenantLock) TTL() time.Duration { return t.ttl }

// Acquire attempts SET NX EX. Returns (true, nil) when the lock is now held
// with `owner` as the value, (false, nil) when held by someone else.
func (t *TenantLock) Acquire(ctx context.Context, key, owner string) (bool, error) {
	ok, err := t.client.SetNX(ctx, key, owner, t.ttl).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// Refresh extends the TTL only if the caller still owns the lock.
const refreshLua = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("EXPIRE", KEYS[1], ARGV[2])
else
  return 0
end
`

func (t *TenantLock) Refresh(ctx context.Context, key, owner string) error {
	secs := int64(t.ttl / time.Second)
	if secs <= 0 {
		secs = 1
	}
	return t.client.Eval(ctx, refreshLua, []string{key}, owner, secs).Err()
}

// Release deletes the lock only if the value matches `owner`.
const releaseLua = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
else
  return 0
end
`

func (t *TenantLock) Release(ctx context.Context, key, owner string) error {
	return t.client.Eval(ctx, releaseLua, []string{key}, owner).Err()
}
