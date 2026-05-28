package listener

import (
	"sync"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Evictor is called once per tenant when the last listener for that
// tenant transitions to Removed. Evictors should drop per-tenant caches
// (monster status mirrors, character keys, account sessions, …) so a
// later re-Add for the same tenant starts from a clean slate.
type Evictor func(t tenant.Model)

var (
	evMu     sync.Mutex
	evictors []Evictor
)

// RegisterEvictor adds fn to the global evictor list. Safe to call from
// init() of any package that holds tenant-scoped state. There is no
// Deregister — evictors are process-lifetime.
func RegisterEvictor(fn Evictor) {
	evMu.Lock()
	defer evMu.Unlock()
	evictors = append(evictors, fn)
}

func fireEvictorsForTenant(t tenant.Model) {
	evMu.Lock()
	snap := make([]Evictor, len(evictors))
	copy(snap, evictors)
	evMu.Unlock()
	for _, fn := range snap {
		fn(t)
	}
}

// SetEvictorsForTest replaces the global evictor list with only the
// provided evictors and restores the previous list at test cleanup.
// Test-only seam: takes *testing.T to make accidental production use
// impossible.
func SetEvictorsForTest(t *testing.T, fns ...Evictor) {
	t.Helper()
	evMu.Lock()
	prev := evictors
	evictors = append([]Evictor(nil), fns...)
	evMu.Unlock()
	t.Cleanup(func() {
		evMu.Lock()
		evictors = prev
		evMu.Unlock()
	})
}
