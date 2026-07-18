package test

import (
	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// CreateDefaultMockTenant creates a new mock tenant with a default ID
func CreateDefaultMockTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}
