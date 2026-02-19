package compartment

import (
	"context"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas-constants/inventory"
	goredis "github.com/redis/go-redis/v9"
)

const (
	lockTTL     = 30 * time.Second
	lockRetry   = 50 * time.Millisecond
	lockTimeout = 10 * time.Second
)

// DistributedMutex provides Lock/Unlock semantics backed by Redis.
// It is returned by lockRegistry.Get and is compatible with call sites
// that previously used *sync.RWMutex (Lock/Unlock only — RLock/RUnlock are not used).
type DistributedMutex struct {
	client *goredis.Client
	key    string
	value  string
}

var unlockScript = goredis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
end
return 0
`)

func (m *DistributedMutex) Lock() {
	m.value = fmt.Sprintf("%d", time.Now().UnixNano())
	deadline := time.Now().Add(lockTimeout)
	for time.Now().Before(deadline) {
		ok, err := m.client.SetNX(context.Background(), m.key, m.value, lockTTL).Result()
		if err == nil && ok {
			return
		}
		time.Sleep(lockRetry)
	}
	// Fallback: force acquire if timeout exceeded (prevents deadlock from crashed holders)
	m.client.Set(context.Background(), m.key, m.value, lockTTL)
}

func (m *DistributedMutex) Unlock() {
	unlockScript.Run(context.Background(), m.client, []string{m.key}, m.value)
}

type lockRegistry struct {
	client *goredis.Client
}

var lr *lockRegistry

func InitLockRegistry(client *goredis.Client) {
	lr = &lockRegistry{client: client}
}

func LockRegistry() *lockRegistry {
	return lr
}

func invLockKey(characterId uint32, inventoryType inventory.Type) string {
	return fmt.Sprintf("invlock:%d:%d", characterId, inventoryType)
}

func (r *lockRegistry) Get(characterId uint32, inventoryType inventory.Type) *DistributedMutex {
	return &DistributedMutex{
		client: r.client,
		key:    invLockKey(characterId, inventoryType),
	}
}
