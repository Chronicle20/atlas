package item

import (
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateEquipmentClassification_SetsJobMask(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdxFull(t, db, ctx, tn.Id(), 1452000, "Wooden Bow", 1, "bow", nil)

	require.NoError(t, UpdateEquipmentClassification(db, ctx, 1452000, "Bw", 4))

	var row testSearchIndexEntity
	require.NoError(t, db.WithContext(ctx).First(&row, "tenant_id = ? AND item_id = ?", tn.Id(), 1452000).Error)
	require.NotNil(t, row.JobMask)
	assert.Equal(t, uint8(4), *row.JobMask)
	assert.Equal(t, "bow", row.Subcategory)
	assert.Equal(t, "Wooden Bow", row.Name)
}

func TestUpdateEquipmentClassification_NoClassRestriction(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdxFull(t, db, ctx, tn.Id(), 1002000, "Snail Shell Helmet", 1, "hat", nil)

	require.NoError(t, UpdateEquipmentClassification(db, ctx, 1002000, "Cp", 0))

	var row testSearchIndexEntity
	require.NoError(t, db.WithContext(ctx).First(&row, "tenant_id = ? AND item_id = ?", tn.Id(), 1002000).Error)
	require.NotNil(t, row.JobMask)
	assert.Equal(t, uint8(0), *row.JobMask)
}

func TestUpdateEquipmentClassification_SlotDisambiguatesEarringVsTop(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdxFull(t, db, ctx, tn.Id(), 1040002, "White Undershirt", 1, "earring", nil)

	require.NoError(t, UpdateEquipmentClassification(db, ctx, 1040002, "Cp", 0))

	var row testSearchIndexEntity
	require.NoError(t, db.WithContext(ctx).First(&row, "tenant_id = ? AND item_id = ?", tn.Id(), 1040002).Error)
	assert.Equal(t, "top", row.Subcategory)
}

func TestUpdateEquipmentClassification_Idempotent(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdxFull(t, db, ctx, tn.Id(), 1452000, "Wooden Bow", 1, "bow", nil)

	require.NoError(t, UpdateEquipmentClassification(db, ctx, 1452000, "Bw", 4))
	require.NoError(t, UpdateEquipmentClassification(db, ctx, 1452000, "Bw", 4))

	var row testSearchIndexEntity
	require.NoError(t, db.WithContext(ctx).First(&row, "tenant_id = ? AND item_id = ?", tn.Id(), 1452000).Error)
	require.NotNil(t, row.JobMask)
	assert.Equal(t, uint8(4), *row.JobMask)
}
