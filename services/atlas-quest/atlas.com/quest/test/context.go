package test

import (
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
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

// CreateTestField creates a default field model for testing
func CreateTestField() field.Model {
	return field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
}

// CreateTestFieldWithMap creates a field model with a specific map ID
func CreateTestFieldWithMap(mapId uint32) field.Model {
	return field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(mapId)).Build()
}
