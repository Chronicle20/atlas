package skill

import (
	"sync"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Cache holds skill data models per tenant. Skill data is immutable for
// the lifetime of a tenant, so entries are never invalidated.
type Key struct {
	Tenant  tenant.Model
	SkillId uint32
}

type Cache struct {
	mutex  sync.RWMutex
	skills map[Key]Model
}

var (
	cache *Cache
	once  sync.Once
)

func GetCache() *Cache {
	once.Do(func() {
		cache = &Cache{skills: make(map[Key]Model)}
	})
	return cache
}

func (c *Cache) Get(t tenant.Model, skillId uint32) (Model, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	m, ok := c.skills[Key{Tenant: t, SkillId: skillId}]
	return m, ok
}

func (c *Cache) Put(t tenant.Model, skillId uint32, m Model) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.skills[Key{Tenant: t, SkillId: skillId}] = m
}
