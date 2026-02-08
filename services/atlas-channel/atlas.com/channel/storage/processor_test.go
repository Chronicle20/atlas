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

func TestStorageData_EmptyAssets(t *testing.T) {
	sd := StorageData{
		Capacity: DefaultStorageCapacity,
		Mesos:    0,
		Assets:   []asset.Model{},
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
