package storage

import (
	"atlas-channel/asset"
	"testing"
	"time"
)

func TestDefaultStorageCapacity(t *testing.T) {
	if DefaultStorageCapacity != 4 {
		t.Errorf("Expected DefaultStorageCapacity to be 4, got %d", DefaultStorageCapacity)
	}
}

func TestBuildReferenceData_Equipable(t *testing.T) {
	result := buildReferenceData(asset.ReferenceTypeEquipable, 1, 12345, 0)
	if result != nil {
		t.Errorf("Expected nil for equipable reference data, got %v", result)
	}
}

func TestBuildReferenceData_Consumable(t *testing.T) {
	quantity := uint32(50)
	ownerId := uint32(12345)
	flag := uint16(1)

	result := buildReferenceData(asset.ReferenceTypeConsumable, quantity, ownerId, flag)
	if result == nil {
		t.Fatal("Expected non-nil result for consumable reference data")
	}

	consumable, ok := result.(asset.ConsumableReferenceData)
	if !ok {
		t.Fatalf("Expected ConsumableReferenceData type, got %T", result)
	}

	if consumable.Quantity() != quantity {
		t.Errorf("Expected quantity %d, got %d", quantity, consumable.Quantity())
	}
	if consumable.OwnerId() != ownerId {
		t.Errorf("Expected ownerId %d, got %d", ownerId, consumable.OwnerId())
	}
	if consumable.Flag() != flag {
		t.Errorf("Expected flag %d, got %d", flag, consumable.Flag())
	}
}

func TestBuildReferenceData_Setup(t *testing.T) {
	quantity := uint32(25)
	ownerId := uint32(54321)
	flag := uint16(2)

	result := buildReferenceData(asset.ReferenceTypeSetup, quantity, ownerId, flag)
	if result == nil {
		t.Fatal("Expected non-nil result for setup reference data")
	}

	setup, ok := result.(asset.SetupReferenceData)
	if !ok {
		t.Fatalf("Expected SetupReferenceData type, got %T", result)
	}

	if setup.Quantity() != quantity {
		t.Errorf("Expected quantity %d, got %d", quantity, setup.Quantity())
	}
	if setup.OwnerId() != ownerId {
		t.Errorf("Expected ownerId %d, got %d", ownerId, setup.OwnerId())
	}
	if setup.Flag() != flag {
		t.Errorf("Expected flag %d, got %d", flag, setup.Flag())
	}
}

func TestBuildReferenceData_Etc(t *testing.T) {
	quantity := uint32(100)
	ownerId := uint32(99999)
	flag := uint16(3)

	result := buildReferenceData(asset.ReferenceTypeEtc, quantity, ownerId, flag)
	if result == nil {
		t.Fatal("Expected non-nil result for etc reference data")
	}

	etc, ok := result.(asset.EtcReferenceData)
	if !ok {
		t.Fatalf("Expected EtcReferenceData type, got %T", result)
	}

	if etc.Quantity() != quantity {
		t.Errorf("Expected quantity %d, got %d", quantity, etc.Quantity())
	}
	if etc.OwnerId() != ownerId {
		t.Errorf("Expected ownerId %d, got %d", ownerId, etc.OwnerId())
	}
	if etc.Flag() != flag {
		t.Errorf("Expected flag %d, got %d", flag, etc.Flag())
	}
}

func TestBuildReferenceData_Cash(t *testing.T) {
	quantity := uint32(10)
	ownerId := uint32(11111)
	flag := uint16(4)

	result := buildReferenceData(asset.ReferenceTypeCash, quantity, ownerId, flag)
	if result == nil {
		t.Fatal("Expected non-nil result for cash reference data")
	}

	cash, ok := result.(asset.CashReferenceData)
	if !ok {
		t.Fatalf("Expected CashReferenceData type, got %T", result)
	}

	if cash.Quantity() != quantity {
		t.Errorf("Expected quantity %d, got %d", quantity, cash.Quantity())
	}
	if cash.OwnerId() != ownerId {
		t.Errorf("Expected ownerId %d, got %d", ownerId, cash.OwnerId())
	}
	if cash.Flag() != flag {
		t.Errorf("Expected flag %d, got %d", flag, cash.Flag())
	}
}

func TestBuildReferenceData_UnknownType(t *testing.T) {
	// Unknown types should default to ETC reference data
	quantity := uint32(5)
	ownerId := uint32(22222)
	flag := uint16(5)

	result := buildReferenceData(asset.ReferenceType("unknown"), quantity, ownerId, flag)
	if result == nil {
		t.Fatal("Expected non-nil result for unknown reference type (should default to ETC)")
	}

	etc, ok := result.(asset.EtcReferenceData)
	if !ok {
		t.Fatalf("Expected EtcReferenceData type for unknown reference type, got %T", result)
	}

	if etc.Quantity() != quantity {
		t.Errorf("Expected quantity %d, got %d", quantity, etc.Quantity())
	}
}

func TestTransformAsset_Consumable(t *testing.T) {
	expiration := time.Now().Add(24 * time.Hour)
	restModel := AssetRestModel{
		Id:            "123",
		TemplateId:    2000000,
		ReferenceId:   456,
		ReferenceType: "consumable",
		InventoryType: 2,
		Slot:          5,
		Quantity:      100,
		OwnerId:       12345,
		Flag:          1,
		Expiration:    expiration,
	}

	result := transformAsset(restModel, 123)

	if result.Id() != 123 {
		t.Errorf("Expected Id 123, got %d", result.Id())
	}
	if result.TemplateId() != 2000000 {
		t.Errorf("Expected TemplateId 2000000, got %d", result.TemplateId())
	}
	if result.ReferenceId() != 456 {
		t.Errorf("Expected ReferenceId 456, got %d", result.ReferenceId())
	}
	if result.ReferenceType() != asset.ReferenceTypeConsumable {
		t.Errorf("Expected ReferenceType consumable, got %s", result.ReferenceType())
	}
	if result.InventoryType() != asset.InventoryTypeUse {
		t.Errorf("Expected InventoryType 2, got %d", result.InventoryType())
	}
	if result.Slot() != 5 {
		t.Errorf("Expected Slot 5, got %d", result.Slot())
	}
}

func TestTransformAsset_Equipable(t *testing.T) {
	restModel := AssetRestModel{
		Id:            "789",
		TemplateId:    1000000,
		ReferenceId:   999,
		ReferenceType: "equipable",
		InventoryType: 1,
		Slot:          -1,
		Quantity:      1,
		OwnerId:       0,
		Flag:          0,
	}

	result := transformAsset(restModel, 789)

	if result.Id() != 789 {
		t.Errorf("Expected Id 789, got %d", result.Id())
	}
	if result.ReferenceType() != asset.ReferenceTypeEquipable {
		t.Errorf("Expected ReferenceType equipable, got %s", result.ReferenceType())
	}
	if result.InventoryType() != asset.InventoryTypeEquip {
		t.Errorf("Expected InventoryType 1, got %d", result.InventoryType())
	}
}

func TestTransformAsset_UnknownReferenceType(t *testing.T) {
	// Unknown reference types should fall back to ETC
	restModel := AssetRestModel{
		Id:            "555",
		TemplateId:    9999999,
		ReferenceId:   111,
		ReferenceType: "unknown_type",
		InventoryType: 4,
		Slot:          10,
		Quantity:      50,
		OwnerId:       33333,
		Flag:          2,
	}

	result := transformAsset(restModel, 555)

	if result.Id() != 555 {
		t.Errorf("Expected Id 555, got %d", result.Id())
	}
	// Unknown types get normalized to ETC
	if result.ReferenceType() != asset.ReferenceTypeEtc {
		t.Errorf("Expected ReferenceType etc for unknown type, got %s", result.ReferenceType())
	}
}

func TestTransformAsset_AllReferenceTypes(t *testing.T) {
	testCases := []struct {
		name         string
		refType      string
		expectedType asset.ReferenceType
	}{
		{"equipable", "equipable", asset.ReferenceTypeEquipable},
		{"consumable", "consumable", asset.ReferenceTypeConsumable},
		{"setup", "setup", asset.ReferenceTypeSetup},
		{"etc", "etc", asset.ReferenceTypeEtc},
		{"cash", "cash", asset.ReferenceTypeCash},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restModel := AssetRestModel{
				Id:            "1",
				TemplateId:    1000000,
				ReferenceId:   1,
				ReferenceType: tc.refType,
				InventoryType: 1,
				Slot:          1,
			}

			result := transformAsset(restModel, 1)
			if result.ReferenceType() != tc.expectedType {
				t.Errorf("Expected ReferenceType %s, got %s", tc.expectedType, result.ReferenceType())
			}
		})
	}
}

func TestStorageData_EmptyAssets(t *testing.T) {
	sd := StorageData{
		Capacity: DefaultStorageCapacity,
		Mesos:    0,
		Assets:   []asset.Model[any]{},
	}

	if sd.Capacity != DefaultStorageCapacity {
		t.Errorf("Expected Capacity %d, got %d", DefaultStorageCapacity, sd.Capacity)
	}
	if sd.Mesos != 0 {
		t.Errorf("Expected Mesos 0, got %d", sd.Mesos)
	}
	if len(sd.Assets) != 0 {
		t.Errorf("Expected empty Assets, got %d items", len(sd.Assets))
	}
}
