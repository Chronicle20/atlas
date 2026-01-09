package storage

import (
	"atlas-channel/asset"
	"testing"
)

func TestDefaultStorageCapacity(t *testing.T) {
	if DefaultStorageCapacity != 4 {
		t.Errorf("Expected DefaultStorageCapacity to be 4, got %d", DefaultStorageCapacity)
	}
}

func TestInventoryTypeFromTemplateId(t *testing.T) {
	testCases := []struct {
		templateId uint32
		expected   asset.InventoryType
	}{
		{1000000, asset.InventoryTypeEquip},
		{2000000, asset.InventoryTypeUse},
		{3000000, asset.InventoryTypeSetup},
		{4000000, asset.InventoryTypeEtc},
		{5000000, asset.InventoryTypeCash},
		{9999999, asset.InventoryTypeEtc}, // Unknown defaults to Etc
	}

	for _, tc := range testCases {
		result := inventoryTypeFromTemplateId(tc.templateId)
		if result != tc.expected {
			t.Errorf("For templateId %d, expected %d but got %d", tc.templateId, tc.expected, result)
		}
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
