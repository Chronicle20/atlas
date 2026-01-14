package pet

import (
	"testing"
)

func TestModelBuilder_Build_Success(t *testing.T) {
	_, err := NewModelBuilder(0, 7000000, 5000017, "Test Pet", 1).Build()
	if err != nil {
		t.Fatalf("Expected successful build, got error: %v", err)
	}
}

func TestModelBuilder_Build_MissingTemplateId(t *testing.T) {
	_, err := NewModelBuilder(0, 7000000, 0, "Test Pet", 1).Build()
	if err == nil {
		t.Fatalf("Expected error for missing templateId, got none")
	}
	if err.Error() != "templateId is required" {
		t.Fatalf("Expected 'templateId is required' error, got: %v", err)
	}
}

func TestModelBuilder_Build_MissingOwnerId(t *testing.T) {
	_, err := NewModelBuilder(0, 7000000, 5000017, "Test Pet", 0).Build()
	if err == nil {
		t.Fatalf("Expected error for missing ownerId, got none")
	}
	if err.Error() != "ownerId is required" {
		t.Fatalf("Expected 'ownerId is required' error, got: %v", err)
	}
}

func TestModelBuilder_Build_MissingName(t *testing.T) {
	_, err := NewModelBuilder(0, 7000000, 5000017, "", 1).Build()
	if err == nil {
		t.Fatalf("Expected error for missing name, got none")
	}
	if err.Error() != "name is required" {
		t.Fatalf("Expected 'name is required' error, got: %v", err)
	}
}

func TestModelBuilder_Build_InvalidLevel_TooLow(t *testing.T) {
	_, err := NewModelBuilder(0, 7000000, 5000017, "Test Pet", 1).SetLevel(0).Build()
	if err == nil {
		t.Fatalf("Expected error for level too low, got none")
	}
	if err.Error() != "level must be between 1 and 30" {
		t.Fatalf("Expected 'level must be between 1 and 30' error, got: %v", err)
	}
}

func TestModelBuilder_Build_InvalidLevel_TooHigh(t *testing.T) {
	_, err := NewModelBuilder(0, 7000000, 5000017, "Test Pet", 1).SetLevel(31).Build()
	if err == nil {
		t.Fatalf("Expected error for level too high, got none")
	}
	if err.Error() != "level must be between 1 and 30" {
		t.Fatalf("Expected 'level must be between 1 and 30' error, got: %v", err)
	}
}

func TestModelBuilder_Build_InvalidFullness(t *testing.T) {
	_, err := NewModelBuilder(0, 7000000, 5000017, "Test Pet", 1).SetFullness(101).Build()
	if err == nil {
		t.Fatalf("Expected error for fullness too high, got none")
	}
	if err.Error() != "fullness must be between 0 and 100" {
		t.Fatalf("Expected 'fullness must be between 0 and 100' error, got: %v", err)
	}
}

func TestModelBuilder_Build_InvalidSlot_TooLow(t *testing.T) {
	_, err := NewModelBuilder(0, 7000000, 5000017, "Test Pet", 1).SetSlot(-2).Build()
	if err == nil {
		t.Fatalf("Expected error for slot too low, got none")
	}
	if err.Error() != "slot must be -1 or between 0 and 2" {
		t.Fatalf("Expected 'slot must be -1 or between 0 and 2' error, got: %v", err)
	}
}

func TestModelBuilder_Build_InvalidSlot_TooHigh(t *testing.T) {
	_, err := NewModelBuilder(0, 7000000, 5000017, "Test Pet", 1).SetSlot(3).Build()
	if err == nil {
		t.Fatalf("Expected error for slot too high, got none")
	}
	if err.Error() != "slot must be -1 or between 0 and 2" {
		t.Fatalf("Expected 'slot must be -1 or between 0 and 2' error, got: %v", err)
	}
}

func TestModelBuilder_Build_ValidBoundaries(t *testing.T) {
	tests := []struct {
		name  string
		build func() (Model, error)
	}{
		{
			name: "level at min (1)",
			build: func() (Model, error) {
				return NewModelBuilder(0, 7000000, 5000017, "Test Pet", 1).SetLevel(1).Build()
			},
		},
		{
			name: "level at max (30)",
			build: func() (Model, error) {
				return NewModelBuilder(0, 7000000, 5000017, "Test Pet", 1).SetLevel(30).Build()
			},
		},
		{
			name: "fullness at min (0)",
			build: func() (Model, error) {
				return NewModelBuilder(0, 7000000, 5000017, "Test Pet", 1).SetFullness(0).Build()
			},
		},
		{
			name: "fullness at max (100)",
			build: func() (Model, error) {
				return NewModelBuilder(0, 7000000, 5000017, "Test Pet", 1).SetFullness(100).Build()
			},
		},
		{
			name: "slot at -1 (unslotted)",
			build: func() (Model, error) {
				return NewModelBuilder(0, 7000000, 5000017, "Test Pet", 1).SetSlot(-1).Build()
			},
		},
		{
			name: "slot at 0 (lead)",
			build: func() (Model, error) {
				return NewModelBuilder(0, 7000000, 5000017, "Test Pet", 1).SetSlot(0).Build()
			},
		},
		{
			name: "slot at 2 (max)",
			build: func() (Model, error) {
				return NewModelBuilder(0, 7000000, 5000017, "Test Pet", 1).SetSlot(2).Build()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.build()
			if err != nil {
				t.Fatalf("Expected successful build for %s, got error: %v", tt.name, err)
			}
		})
	}
}
