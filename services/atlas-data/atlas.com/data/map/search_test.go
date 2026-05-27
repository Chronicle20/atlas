package _map

import (
	"context"
	"strconv"
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedIndex(t *testing.T, db *gorm.DB, ctx context.Context, tenantId uuid.UUID, id uint32, name, street string) {
	t.Helper()
	row := testSearchIndexEntity{
		TenantId: tenantId, MapId: id, Name: name, StreetName: street, UpdatedAt: time.Now(),
	}
	// Bypass the tenant create-callback's auto-injection: this fixture deliberately
	// writes the supplied tenantId (which may be uuid.Nil to seed a "global" row).
	require.NoError(t, db.WithContext(database.WithoutTenantFilter(ctx)).Create(&row).Error)
}

func TestSearch_ExactIdFirst(t *testing.T) {
	db := setupStorageTestDB(t)
	l, _ := test.NewNullLogger()

	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	seedIndex(t, db, ctx, tn.Id(), 200, "Alpha 200", "Street A")
	seedIndex(t, db, ctx, tn.Id(), 201, "Alpha 201", "Street A")
	seedIndex(t, db, ctx, tn.Id(), 202, "Alpha 202", "Street A")

	res, err := SearchByQuery(l, db)(ctx)("201", 50)
	require.NoError(t, err)
	require.NotEmpty(t, res)
	assert.Equal(t, uint32(201), res[0].Id, "exact-ID match should be first")
}

func TestSearch_SubstringOnName(t *testing.T) {
	db := setupStorageTestDB(t)
	l, _ := test.NewNullLogger()

	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	seedIndex(t, db, ctx, tn.Id(), 100000000, "Henesys", "Victoria Road")
	seedIndex(t, db, ctx, tn.Id(), 101000000, "Ellinia", "Victoria Road")

	res, err := SearchByQuery(l, db)(ctx)("nesys", 50)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "Henesys", res[0].Name)
}

func TestSearch_SubstringOnStreet_CaseInsensitive(t *testing.T) {
	db := setupStorageTestDB(t)
	l, _ := test.NewNullLogger()

	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	seedIndex(t, db, ctx, tn.Id(), 1, "One", "Perion Street")
	seedIndex(t, db, ctx, tn.Id(), 2, "Two", "Kerning City")

	res, err := SearchByQuery(l, db)(ctx)("PERION", 50)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, uint32(1), res[0].Id)
}

func TestSearch_LimitEnforced(t *testing.T) {
	db := setupStorageTestDB(t)
	l, _ := test.NewNullLogger()

	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	for i := 0; i < 60; i++ {
		seedIndex(t, db, ctx, tn.Id(), uint32(3000+i), "Testmap "+strconv.Itoa(i), "Somewhere")
	}

	res, err := SearchByQuery(l, db)(ctx)("testmap", 50)
	require.NoError(t, err)
	assert.Len(t, res, 50)
}

func TestSearch_SinglePartition_TenantOwnsDataset(t *testing.T) {
	db := setupStorageTestDB(t)
	l, _ := test.NewNullLogger()

	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	seedIndex(t, db, ctx, tn.Id(), 100, "TenantMap", "TenantStreet")
	// global rows are shadowed because the active tenant has rows
	seedIndex(t, db, ctx, uuid.Nil, 100, "GlobalMap", "GlobalStreet")
	seedIndex(t, db, ctx, uuid.Nil, 101, "ExtraMap", "Road")

	res, err := SearchByQuery(l, db)(ctx)("map", 50)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "TenantMap", res[0].Name)
}

func TestSearch_SinglePartition_ZeroRowTenantFallsBack(t *testing.T) {
	db := setupStorageTestDB(t)
	l, _ := test.NewNullLogger()

	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)
	_ = tn

	seedIndex(t, db, ctx, uuid.Nil, 100, "GlobalMap", "GlobalStreet")
	seedIndex(t, db, ctx, uuid.Nil, 101, "ExtraMap", "Road")

	res, err := SearchByQuery(l, db)(ctx)("map", 50)
	require.NoError(t, err)
	require.Len(t, res, 2)
}
