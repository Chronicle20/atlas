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

	require.NoError(t, UpdateEquipmentClassification(db, ctx, 1452000, 4, false))

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

	require.NoError(t, UpdateEquipmentClassification(db, ctx, 1002000, 0, false))

	var row testSearchIndexEntity
	require.NoError(t, db.WithContext(ctx).First(&row, "tenant_id = ? AND item_id = ?", tn.Id(), 1002000).Error)
	require.NotNil(t, row.JobMask)
	assert.Equal(t, uint8(0), *row.JobMask)
}

func TestUpdateEquipmentClassification_CashOverridesCompartment(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	// 1052021 is a cash overall — id prefix says equipment, but the WZ Cash flag puts it in Cash.
	seedIdxFull(t, db, ctx, tn.Id(), 1052021, "Cash Overall", uint8(CompartmentEquipment), "overall", nil)

	require.NoError(t, UpdateEquipmentClassification(db, ctx, 1052021, 0, true))

	var row testSearchIndexEntity
	require.NoError(t, db.WithContext(ctx).First(&row, "tenant_id = ? AND item_id = ?", tn.Id(), 1052021).Error)
	assert.Equal(t, uint8(CompartmentCash), row.Compartment)
}

func TestUpdateEquipmentClassification_Idempotent(t *testing.T) {
	db := setupSearchTestDB(t)
	ctx := tenant.WithContext(context.Background(), newSearchTenant(t))
	tn := tenant.MustFromContext(ctx)

	seedIdxFull(t, db, ctx, tn.Id(), 1452000, "Wooden Bow", 1, "bow", nil)

	require.NoError(t, UpdateEquipmentClassification(db, ctx, 1452000, 4, false))
	require.NoError(t, UpdateEquipmentClassification(db, ctx, 1452000, 4, false))

	var row testSearchIndexEntity
	require.NoError(t, db.WithContext(ctx).First(&row, "tenant_id = ? AND item_id = ?", tn.Id(), 1452000).Error)
	require.NotNil(t, row.JobMask)
	assert.Equal(t, uint8(4), *row.JobMask)
}
