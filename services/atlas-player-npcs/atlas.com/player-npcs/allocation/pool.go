package allocation

import (
	"sync"

	"github.com/google/uuid"
)

// Lookup answers whether a candidate script id has a template, and whether
// that template's imitate flag is set. It is supplied by the caller (backed
// by atlas-data in production) so this package stays free of an HTTP
// client.
type Lookup func(id uint32) (exists bool, imitate bool, err error)

// BuildUsablePool calls lookup for every id in [PoolMin, PoolMax] and
// returns the subset that is unallocatable-safe: the template must exist
// AND its imitate flag must be set (PRD §10 acceptance criteria). Either
// condition failing drops the id from the set; a lookup error aborts the
// build and is surfaced to the caller.
func BuildUsablePool(lookup Lookup) (map[uint32]bool, error) {
	usable := make(map[uint32]bool)
	for id := PoolMin; id <= PoolMax; id++ {
		exists, imitate, err := lookup(id)
		if err != nil {
			return nil, err
		}
		if exists && imitate {
			usable[id] = true
		}
	}
	return usable, nil
}

// poolCache holds the per-tenant usable set for the process lifetime.
// atlas-data projects WZ read-only, so the set cannot go stale without a
// restart (design §4.2) — it is built once, lazily, per tenant.
var (
	poolCacheMu sync.Mutex
	poolCache   = make(map[uuid.UUID]map[uint32]bool)
)

// UsablePoolFor returns the cached usable set for tenantId, building and
// caching it via BuildUsablePool on first use.
func UsablePoolFor(tenantId uuid.UUID, lookup Lookup) (map[uint32]bool, error) {
	poolCacheMu.Lock()
	defer poolCacheMu.Unlock()

	if pool, ok := poolCache[tenantId]; ok {
		return pool, nil
	}

	pool, err := BuildUsablePool(lookup)
	if err != nil {
		return nil, err
	}
	poolCache[tenantId] = pool
	return pool, nil
}
