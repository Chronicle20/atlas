package item

import (
	"atlas-query-aggregator/item/mock"
	"testing"
)

func TestProcessorMock_GetSlotMax_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetSlotMaxFunc: func(itemId uint32) uint32 {
			switch itemId {
			case 2000001: // Consumable
				return 100
			case 1000001: // Equipment
				return 1
			case 4000001: // ETC item
				return 200
			default:
				return 100
			}
		},
	}

	// Test consumable
	slotMax := mockProcessor.GetSlotMax(2000001)
	if slotMax != 100 {
		t.Errorf("Expected slotMax=100 for consumable, got %d", slotMax)
	}

	// Test equipment (never stacks)
	slotMax = mockProcessor.GetSlotMax(1000001)
	if slotMax != 1 {
		t.Errorf("Expected slotMax=1 for equipment, got %d", slotMax)
	}

	// Test ETC item
	slotMax = mockProcessor.GetSlotMax(4000001)
	if slotMax != 200 {
		t.Errorf("Expected slotMax=200 for ETC item, got %d", slotMax)
	}
}

func TestProcessorMock_GetSlotMax_DefaultValue(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetSlotMaxFunc: func(itemId uint32) uint32 {
			return 100 // Default for unknown items
		},
	}

	slotMax := mockProcessor.GetSlotMax(9999999)
	if slotMax != 100 {
		t.Errorf("Expected slotMax=100 for unknown item, got %d", slotMax)
	}
}

func TestProcessorMock_DefaultBehavior(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{}

	// Test default GetSlotMax returns 100
	slotMax := mockProcessor.GetSlotMax(2000001)
	if slotMax != 100 {
		t.Errorf("Expected default slotMax=100, got %d", slotMax)
	}
}

func TestGetDefaultSlotMax(t *testing.T) {
	tests := []struct {
		name     string
		itemId   uint32
		expected uint32
	}{
		{"consumable item", 2000001, 100},
		{"equipment item", 1000001, 1},
		{"etc item", 4000001, 100},
		{"setup item", 3000001, 100},
		{"cash item", 5000001, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slotMax := GetDefaultSlotMax(tt.itemId)
			if slotMax != tt.expected {
				t.Errorf("Expected slotMax=%d for item %d, got %d", tt.expected, tt.itemId, slotMax)
			}
		})
	}
}
