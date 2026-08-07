package writer

import (
	"sync"

	"github.com/google/uuid"

	opcodes "github.com/Chronicle20/atlas/libs/atlas-opcodes"
)

// tenantWriterOptions holds each tenant's writer options tables so domain
// logic outside an encode path can resolve the same config-driven wire
// values writers use (DOM-25) — e.g. the battleship mount arm resolving the
// vehicle id or the damage handler resolving the gauge pseudo-skill id from
// the tenant's socket configuration. Populated when the tenant's listener is
// built; evicted alongside the tenant's other caches.
var (
	tenantWriterOptionsMu sync.RWMutex
	tenantWriterOptions   = map[uuid.UUID]map[string]map[string]interface{}{}
)

// RegisterTenantWriterOptions records the options table of every writer in
// the tenant's socket configuration. Writers without options are skipped.
func RegisterTenantWriterOptions(tenantId uuid.UUID, writers []opcodes.WriterConfig) {
	tables := make(map[string]map[string]interface{})
	for _, wc := range writers {
		if len(wc.Options) > 0 {
			tables[wc.Writer] = wc.Options
		}
	}
	tenantWriterOptionsMu.Lock()
	defer tenantWriterOptionsMu.Unlock()
	tenantWriterOptions[tenantId] = tables
}

// TenantWriterOptions returns the named writer's options table for the
// tenant. ok=false when the tenant is unregistered or the writer has no
// options configured.
func TenantWriterOptions(tenantId uuid.UUID, writerName string) (map[string]interface{}, bool) {
	tenantWriterOptionsMu.RLock()
	defer tenantWriterOptionsMu.RUnlock()
	tables, ok := tenantWriterOptions[tenantId]
	if !ok {
		return nil, false
	}
	opts, ok := tables[writerName]
	return opts, ok
}

// EvictTenantWriterOptions drops the tenant's tables.
func EvictTenantWriterOptions(tenantId uuid.UUID) {
	tenantWriterOptionsMu.Lock()
	defer tenantWriterOptionsMu.Unlock()
	delete(tenantWriterOptions, tenantId)
}
