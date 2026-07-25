package shopscanner

import (
	"sync"

	"github.com/google/uuid"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Key struct {
	Tenant      tenant.Model
	CharacterId uint32
}

// SearchEntry remembers a character's most recent executed owl search — the
// OWL_WARP handler validates the clicked result against it.
type SearchEntry struct {
	ItemId uint32
}

// PendingEntry marks a warp-then-enter in flight: set when OWL_WARP passes
// validation, consumed on VisitorEntered/CapacityFull, dropped on
// arrival-map mismatch and session destroy.
type PendingEntry struct {
	ShopId  uuid.UUID
	OwnerId uint32
	MapId   _map.Id
}

type Registry struct {
	mutex      sync.RWMutex
	lastSearch map[Key]SearchEntry
	pending    map[Key]PendingEntry
}

var (
	registry *Registry
	once     sync.Once
)

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.lastSearch = make(map[Key]SearchEntry)
		registry.pending = make(map[Key]PendingEntry)
	})
	return registry
}

func (r *Registry) SetLastSearch(t tenant.Model, characterId uint32, itemId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.lastSearch[Key{Tenant: t, CharacterId: characterId}] = SearchEntry{ItemId: itemId}
}

func (r *Registry) GetLastSearch(t tenant.Model, characterId uint32) (SearchEntry, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	e, ok := r.lastSearch[Key{Tenant: t, CharacterId: characterId}]
	return e, ok
}

func (r *Registry) SetPending(t tenant.Model, characterId uint32, e PendingEntry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.pending[Key{Tenant: t, CharacterId: characterId}] = e
}

func (r *Registry) GetPending(t tenant.Model, characterId uint32) (PendingEntry, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	e, ok := r.pending[Key{Tenant: t, CharacterId: characterId}]
	return e, ok
}

func (r *Registry) RemovePending(t tenant.Model, characterId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.pending, Key{Tenant: t, CharacterId: characterId})
}

// ClearCharacter drops all scanner state for a character (session destroy).
func (r *Registry) ClearCharacter(t tenant.Model, characterId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	k := Key{Tenant: t, CharacterId: characterId}
	delete(r.lastSearch, k)
	delete(r.pending, k)
}
