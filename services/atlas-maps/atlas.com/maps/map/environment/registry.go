package environment

import (
	"slices"
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type FieldKey struct {
	Tenant tenant.Model
	Field  field.Model
}

// ObjectEntry tracks an object's current State and the DefaultState the map
// declares for it. atlas-maps only learns an object exists once something sets
// it, so it never observes the pre-change value -- DefaultState is resolved
// from atlas-data on first track and is what a reset restores.
type ObjectEntry struct {
	Kind         field.ObjectKind
	Name         string
	State        uint32
	DefaultState uint32
}

type Registry struct {
	mutex   sync.RWMutex
	entries map[FieldKey][]ObjectEntry
}

var (
	registry *Registry
	once     sync.Once
)

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.entries = make(map[FieldKey][]ObjectEntry)
	})
	return registry
}

// Set replaces the entry for (entry.Kind, entry.Name) in place, preserving its
// original index, or appends it when it is new. Insertion order is preserved
// because replay order is observable to the client.
func (r *Registry) Set(key FieldKey, entry ObjectEntry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	existing := r.entries[key]
	for i, e := range existing {
		if e.Kind == entry.Kind && e.Name == entry.Name {
			existing[i] = entry
			r.entries[key] = existing
			return
		}
	}
	r.entries[key] = append(existing, entry)
}

// DefaultState returns the declared default already retained for a tracked
// object, so a re-set does not re-resolve it from atlas-data.
func (r *Registry) DefaultState(key FieldKey, kind field.ObjectKind, name string) (uint32, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	for _, e := range r.entries[key] {
		if e.Kind == kind && e.Name == name {
			return e.DefaultState, true
		}
	}
	return 0, false
}

// Get returns a copy of the field's entries. ObjectEntry is a value type with
// no reference fields, so slices.Clone is a full deep copy and a concurrent
// Set cannot tear the caller's view.
func (r *Registry) Get(key FieldKey) []ObjectEntry {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	e, ok := r.entries[key]
	if !ok {
		return make([]ObjectEntry, 0)
	}
	return slices.Clone(e)
}

// Clear removes the key entirely and returns what it held, so a field with no
// tracked state occupies no map entry.
func (r *Registry) Clear(key FieldKey) []ObjectEntry {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	e, ok := r.entries[key]
	delete(r.entries, key)
	if !ok {
		return make([]ObjectEntry, 0)
	}
	return e
}

func (r *Registry) Delete(key FieldKey) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.entries, key)
}
