package compartment

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

// Get builds a distributed mutex whose key is tenant-scoped: atlas.Lock
// itself is a process-wide singleton (namespace "inventory" only), so
// without the tenant segment two tenants' characters sharing the same
// characterId/inventoryType pair would contend for the same Redis key once
// a single atlas-inventory process serves more than one tenant (sparse
// ephemeral environments, D1).
func (r *lockRegistry) Get(t tenant.Model, characterId uint32, inventoryType inventory.Type) *DistributedMutex {
	return &DistributedMutex{
		lock: r.lock,
		key:  atlas.CompositeKey(atlas.TenantKey(t), fmt.Sprintf("%d:%d", characterId, inventoryType)),
	}
}
