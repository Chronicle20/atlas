package drop

import (
	"testing"
)

func TestBuilder_Build_Success(t *testing.T) {
	m, err := NewBuilder().Build()
	if err != nil {
		t.Errorf("Expected no error for default build, got %v", err)
	}
	if m.MinimumQuantity() != 1 {
		t.Errorf("Expected default minQty 1, got %d", m.MinimumQuantity())
	}
	if m.MaximumQuantity() != 1 {
		t.Errorf("Expected default maxQty 1, got %d", m.MaximumQuantity())
	}
}

func TestBuilder_ValidQuantities(t *testing.T) {
	testCases := []struct {
		name string
		min  uint32
		max  uint32
	}{
		{"min equals max", 5, 5},
		{"min less than max", 1, 10},
		{"zero min", 0, 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewBuilder().
				SetMinimumQuantity(tc.min).
				SetMaximumQuantity(tc.max).
				Build()
			if err != nil {
				t.Errorf("Expected no error for min=%d, max=%d, got %v", tc.min, tc.max, err)
			}
		})
	}
}

func TestBuilder_InvalidQuantities(t *testing.T) {
	_, err := NewBuilder().
		SetMinimumQuantity(10).
		SetMaximumQuantity(5).
		Build()

	if err == nil {
		t.Errorf("Expected error when minimumQuantity > maximumQuantity")
	}

	expectedMsg := "minimumQuantity cannot be greater than maximumQuantity"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}
