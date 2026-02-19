package weather

import (
	"sync"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-tenant"
)

type FieldKey struct {
	Tenant tenant.Model
	Field  field.Model
}

type WeatherEntry struct {
	ItemId    uint32
	Message   string
	ExpiresAt time.Time
}

type Registry struct {
	mutex   sync.RWMutex
	entries map[FieldKey]WeatherEntry
}

var registry *Registry
var once sync.Once

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.entries = make(map[FieldKey]WeatherEntry)
	})
	return registry
}

func (r *Registry) Set(key FieldKey, entry WeatherEntry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.entries[key] = entry
}

func (r *Registry) Get(key FieldKey) (WeatherEntry, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	e, ok := r.entries[key]
	return e, ok
}

func (r *Registry) Delete(key FieldKey) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.entries, key)
}

type ExpiredEntry struct {
	Key   FieldKey
	Entry WeatherEntry
}

func (r *Registry) GetExpired() []ExpiredEntry {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	now := time.Now()
	result := make([]ExpiredEntry, 0)
	for k, e := range r.entries {
		if now.After(e.ExpiresAt) {
			result = append(result, ExpiredEntry{Key: k, Entry: e})
		}
	}
	return result
}

func GetExpired() []ExpiredEntry {
	return getRegistry().GetExpired()
}

func DeleteEntry(key FieldKey) {
	getRegistry().Delete(key)
}
