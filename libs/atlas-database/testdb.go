package database

import (
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewInMemoryTenantDB returns a fresh sqlite-in-memory *gorm.DB with the tenant
// callbacks registered and every supplied Migrator applied. Use this in provider
// tests instead of hand-rolling sqlite setup. A discard logger is attached so
// callback warnings do not spam test output.
func NewInMemoryTenantDB(t *testing.T, migrations ...Migrator) *gorm.DB {
	t.Helper()
	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	registerTenantCallbacks(l, db)
	for _, m := range migrations {
		require.NoError(t, m(db))
	}
	return db
}

// TenantContext returns a context carrying a tenant with the supplied id (GMS / v83 / region 1).
// Use this in tests to scope a query to a specific tenant.
func TenantContext(id uuid.UUID) context.Context {
	t, _ := tenant.Create(id, "GMS", 83, 1)
	return tenant.WithContext(context.Background(), t)
}
