package session

import "github.com/google/uuid"

// ClearRegistryForTenant clears all sessions for a specific tenant in the registry.
// This function is intended for use in tests only.
func ClearRegistryForTenant(tenantId uuid.UUID) {
	r := getRegistry()
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.sessionRegistry, tenantId)
}

// AddSessionToRegistry adds a session directly to the registry.
// This function is intended for use in tests only.
func AddSessionToRegistry(tenantId uuid.UUID, s Model) {
	getRegistry().Add(tenantId, s)
}
