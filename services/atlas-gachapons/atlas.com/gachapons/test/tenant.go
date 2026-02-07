package test

import (
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

// TestTenantId is a fixed UUID used for all tests
var TestTenantId = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// CreateDefaultMockTenant creates a new mock tenant with a fixed ID for testing
func CreateDefaultMockTenant() tenant.Model {
	t, _ := tenant.Create(TestTenantId, "GMS", 83, 1)
	return t
}
