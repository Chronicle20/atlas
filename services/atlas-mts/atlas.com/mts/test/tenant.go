package test

import (
	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestTenantId is a fixed UUID used for all tests
var TestTenantId = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// CreateDefaultMockTenant creates a new mock tenant with a fixed ID for testing
func CreateDefaultMockTenant() tenant.Model {
	t, _ := tenant.Create(TestTenantId, "GMS", 83, 1)
	return t
}
