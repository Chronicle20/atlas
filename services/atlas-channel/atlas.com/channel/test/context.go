package test

import (
	"context"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

// DefaultTenantId is the default tenant ID used in tests
var DefaultTenantId = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// CreateTestContext creates a context with a mock tenant for testing
func CreateTestContext() context.Context {
	return CreateTestContextWithTenant(DefaultTenantId)
}

// CreateTestContextWithTenant creates a context with a specific tenant ID
func CreateTestContextWithTenant(tenantId uuid.UUID) context.Context {
	t, _ := tenant.Create(tenantId, "GMS", 83, 1)
	return tenant.WithContext(context.Background(), t)
}

// CreateDefaultMockTenant creates a default mock tenant for testing
func CreateDefaultMockTenant() tenant.Model {
	t, _ := tenant.Create(DefaultTenantId, "GMS", 83, 1)
	return t
}

// CreateMockTenant creates a mock tenant with a specific ID for testing
func CreateMockTenant(tenantId uuid.UUID) tenant.Model {
	t, _ := tenant.Create(tenantId, "GMS", 83, 1)
	return t
}
