package asset

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetData_ScannerValuer_Roundtrip(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	equipped := now.Add(-24 * time.Hour)

	original := AssetData{
		Expiration:     now,
		CreatedAt:      now,
		Quantity:       50,
		OwnerId:        1000,
		Flag:           3,
		Rechargeable:   100,
		Strength:       10,
		Dexterity:      12,
		Intelligence:   8,
		Luck:           15,
		Hp:             200,
		Mp:             150,
		WeaponAttack:   45,
		MagicAttack:    30,
		WeaponDefense:  20,
		MagicDefense:   25,
		Accuracy:       5,
		Avoidability:   3,
		Hands:          2,
		Speed:          10,
		Jump:           5,
		Slots:          7,
		LevelType:      1,
		Level:          3,
		Experience:     500,
		HammersApplied: 2,
		EquippedSince:  &equipped,
		CashId:         9876543210,
		CommodityId:    5000,
		PurchaseBy:     2000,
		PetId:          3000,
	}

	val, err := original.Value()
	require.NoError(t, err)

	var restored AssetData
	err = restored.Scan(val)
	require.NoError(t, err)

	assert.Equal(t, original.Quantity, restored.Quantity)
	assert.Equal(t, original.Flag, restored.Flag)
	assert.Equal(t, original.Strength, restored.Strength)
	assert.Equal(t, original.Dexterity, restored.Dexterity)
	assert.Equal(t, original.Intelligence, restored.Intelligence)
	assert.Equal(t, original.Luck, restored.Luck)
	assert.Equal(t, original.Hp, restored.Hp)
	assert.Equal(t, original.Mp, restored.Mp)
	assert.Equal(t, original.WeaponAttack, restored.WeaponAttack)
	assert.Equal(t, original.MagicAttack, restored.MagicAttack)
	assert.Equal(t, original.WeaponDefense, restored.WeaponDefense)
	assert.Equal(t, original.MagicDefense, restored.MagicDefense)
	assert.Equal(t, original.Accuracy, restored.Accuracy)
	assert.Equal(t, original.Avoidability, restored.Avoidability)
	assert.Equal(t, original.Hands, restored.Hands)
	assert.Equal(t, original.Speed, restored.Speed)
	assert.Equal(t, original.Jump, restored.Jump)
	assert.Equal(t, original.Slots, restored.Slots)
	assert.Equal(t, original.LevelType, restored.LevelType)
	assert.Equal(t, original.Level, restored.Level)
	assert.Equal(t, original.Experience, restored.Experience)
	assert.Equal(t, original.HammersApplied, restored.HammersApplied)
	assert.Equal(t, original.CashId, restored.CashId)
	assert.Equal(t, original.CommodityId, restored.CommodityId)
	assert.Equal(t, original.PurchaseBy, restored.PurchaseBy)
	assert.Equal(t, original.PetId, restored.PetId)
	assert.Equal(t, original.Rechargeable, restored.Rechargeable)
	assert.Equal(t, original.OwnerId, restored.OwnerId)
	require.NotNil(t, restored.EquippedSince)
	assert.True(t, original.EquippedSince.Equal(*restored.EquippedSince))
	assert.True(t, original.Expiration.Equal(restored.Expiration))
	assert.True(t, original.CreatedAt.Equal(restored.CreatedAt))
}

func TestAssetData_Scan_Nil(t *testing.T) {
	var a AssetData
	err := a.Scan(nil)
	assert.NoError(t, err)
	assert.Equal(t, AssetData{}, a)
}

func TestAssetData_Scan_InvalidType(t *testing.T) {
	var a AssetData
	err := a.Scan(42)
	assert.Error(t, err)
}

func TestAssetData_WithQuantity(t *testing.T) {
	original := AssetData{Quantity: 10, Flag: 5, Strength: 20}
	updated := original.WithQuantity(50)

	assert.Equal(t, uint32(50), updated.Quantity)
	assert.Equal(t, uint16(5), updated.Flag)
	assert.Equal(t, uint16(20), updated.Strength)
	// Original unchanged.
	assert.Equal(t, uint32(10), original.Quantity)
}
