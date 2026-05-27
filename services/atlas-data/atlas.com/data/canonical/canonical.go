// Package canonical exposes constants for the reserved canonical-scope identity.
//
// The canonical tenant UUID is a sentinel value used to anchor cross-tenant
// shared content (canonical baseline rows in the documents+search-index tables,
// shared MinIO assets, etc.). It is never a real tenant — destructive
// operations against it must be refused at the handler boundary.
package canonical

// TenantUUID is the reserved sentinel UUID for canonical-scope rows. Cannot be
// used as a real tenant id; purge handlers must refuse it.
const TenantUUID = "00000000-0000-0000-0000-000000000000"
