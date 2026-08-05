package tenant

import (
	"strings"

	"github.com/google/uuid"
)

// DerivedId returns a stable UUIDv5 for a tenant-scoped entity, using the
// tenant id as the namespace and the "/"-joined parts as the name.
//
// It exists so a configuration entry identified by a human slug can have
// one deterministic UUID that every service computes identically: stable
// across repeated loads, replicas, restarts and re-seeds; different per
// tenant for the same slug; and different across resource types that
// happen to share a slug.
//
// This formula is load-bearing. It keys the atlas-transports Redis route
// registries, so changing it re-keys every entry and silently duplicates
// the deployed registry. libs/atlas-tenant/id_test.go pins known vectors
// precisely to make such a change fail loudly.
func DerivedId(tenantId uuid.UUID, parts ...string) uuid.UUID {
	return uuid.NewSHA1(tenantId, []byte(strings.Join(parts, "/")))
}
