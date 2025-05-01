package test

import (
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

// CreateDefaultMockTenant creates a new mock tenant with a default ID
func CreateDefaultMockTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}
