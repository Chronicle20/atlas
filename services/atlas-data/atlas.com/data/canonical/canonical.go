// Package canonical exposes constants for the reserved canonical-scope identity.
//
// The canonical tenant UUID is a sentinel value used to anchor cross-tenant
// shared content (canonical baseline rows in the documents+search-index tables,
// shared MinIO assets, etc.). It is never a real tenant — destructive
// operations against it must be refused at the handler boundary.
package canonical

import (
	"fmt"

	"github.com/google/uuid"
)

// TenantUUID is the reserved sentinel UUID for canonical-scope rows. Cannot be
// used as a real tenant id; purge handlers must refuse it.
const TenantUUID = "00000000-0000-0000-0000-000000000000"

// Namespace is the UUID v5 namespace used to derive all version-scoped
// canonical tenant ids via TenantId.
//
// WARNING: This value MUST NOT change once any canonical rows exist in any
// environment. Changing it orphans every canonical row (they would no longer
// match the id returned by TenantId) across every deployment — requiring a
// full re-ingest of all canonical data.
var Namespace = uuid.NewSHA1(uuid.NameSpaceURL, []byte("https://atlas-data/canonical"))

// TenantId returns a deterministic UUID v5 that identifies the canonical
// baseline tenant for a given region and client version (major.minor).
// Two calls with identical arguments always return the same UUID.
// Different region/major/minor combinations return distinct UUIDs.
func TenantId(region string, major, minor uint16) uuid.UUID {
	return uuid.NewSHA1(Namespace, []byte(fmt.Sprintf("canonical:%s:%d.%d", region, major, minor)))
}

// IsCanonical reports whether id equals the canonical tenant id for the given
// region and version. It is the inverse of TenantId: a round-trip through
// TenantId and IsCanonical always returns true.
func IsCanonical(id uuid.UUID, region string, major, minor uint16) bool {
	return id == TenantId(region, major, minor)
}
