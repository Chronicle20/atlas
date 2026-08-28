package backeffect

import (
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type FieldKey struct {
	Tenant tenant.Model
	Field  field.Model
}

type BackEffectEntry struct {
	Effect  byte
	FieldId uint32
	PageId  byte
	// Duration is the client's fade length in milliseconds, as sent in the
	// SET_BACK_EFFECT packet. It is deliberately not an expiry: there is no
	// reaper for back-effect entries, unlike the jukebox registry's
	// ExpiresAt. A CLEAR_BACK_EFFECT command is what removes an entry.
	Duration uint32
}

type Registry struct {
	mutex   sync.RWMutex
	entries map[FieldKey][]BackEffectEntry
}

var (
	registry *Registry
	once     sync.Once
)

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.entries = make(map[FieldKey][]BackEffectEntry)
	})
	return registry
}

func (r *Registry) Set(key FieldKey, entry BackEffectEntry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	entries := r.entries[key]
	for i, e := range entries {
		if e.PageId == entry.PageId {
			entries[i] = entry
			r.entries[key] = entries
			return
		}
	}
	r.entries[key] = append(entries, entry)
}

func (r *Registry) Get(key FieldKey) []BackEffectEntry {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	entries := r.entries[key]
	result := make([]BackEffectEntry, len(entries))
	copy(result, entries)
	return result
}

func (r *Registry) Clear(key FieldKey) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	_, ok := r.entries[key]
	if !ok {
		return false
	}
	delete(r.entries, key)
	return true
}
