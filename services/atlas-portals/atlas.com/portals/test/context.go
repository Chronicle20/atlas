package test

import (
	"context"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// CreateTestContext creates a context with a mock tenant for testing
func CreateTestContext() context.Context {
	return tenant.WithContext(context.Background(), CreateDefaultMockTenant())
}
