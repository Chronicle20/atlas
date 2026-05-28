package compartment

import (
	"context"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
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
	lock  *atlas.Lock
	key   string
	value string
}

func (m *DistributedMutex) Lock() {
	m.value = fmt.Sprintf("%d", time.Now().UnixNano())
	deadline := time.Now().Add(lockTimeout)
	for time.Now().Before(deadline) {
		ok, err := m.lock.AcquireWithToken(context.Background(), m.key, m.value, lockTTL)
		if err == nil && ok {
			return
		}
		time.Sleep(lockRetry)
	}
	// Fallback: force acquire if timeout exceeded (prevents deadlock from crashed holders)
	_ = m.lock.ForceAcquire(context.Background(), m.key, m.value, lockTTL)
}

func (m *DistributedMutex) Unlock() {
	_, _ = m.lock.ReleaseToken(context.Background(), m.key, m.value)
}

type lockRegistry struct {
	lock *atlas.Lock
}

var lr *lockRegistry

func InitLockRegistry(client *goredis.Client) {
	lr = &lockRegistry{lock: atlas.NewLockWithTTL(client, "inventory", lockTTL)}
}

func LockRegistry() *lockRegistry {
	return lr
}

func (r *lockRegistry) Get(characterId uint32, inventoryType inventory.Type) *DistributedMutex {
	return &DistributedMutex{
		lock: r.lock,
		key:  fmt.Sprintf("%d:%d", characterId, inventoryType),
	}
}
