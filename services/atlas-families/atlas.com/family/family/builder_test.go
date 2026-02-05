package family

import (
	"testing"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

func TestBuilder_FluentInterface(t *testing.T) {
	characterId := uint32(12345)
	tenantId := uuid.New()
	level := uint16(50)
	worldId := world.Id(1)

	// Test fluent interface with method chaining
	member, err := NewBuilder(characterId, tenantId, level, worldId).
		SetSeniorId(99999).
		SetJuniorIds([]uint32{11111, 22222}).
		SetRep(1000).
		SetDailyRep(100).
		Build()

	if err != nil {
		t.Fatalf("Fluent builder failed: %v", err)
	}

	// Verify all fields were set correctly
	if member.CharacterId() != characterId {
		t.Errorf("Expected CharacterId %d, got %d", characterId, member.CharacterId())
	}

	if member.SeniorId() == nil || *member.SeniorId() != 99999 {
		seniorId := uint32(0)
		if member.SeniorId() != nil {
			seniorId = *member.SeniorId()
		}
		t.Errorf("Expected SeniorId %d, got %d", 99999, seniorId)
	}

	if len(member.JuniorIds()) != 2 {
		t.Errorf("Expected 2 juniors, got %d", len(member.JuniorIds()))
	}

	if member.Rep() != 1000 {
		t.Errorf("Expected Rep %d, got %d", 1000, member.Rep())
	}

	if member.DailyRep() != 100 {
		t.Errorf("Expected DailyRep %d, got %d", 100, member.DailyRep())
	}
}

func TestBuilder_BusinessLogicMethods(t *testing.T) {
	characterId := uint32(12345)
	tenantId := uuid.New()
	level := uint16(50)
	worldId := world.Id(1)

	builder := NewBuilder(characterId, tenantId, level, worldId)

	t.Run("AddJunior", func(t *testing.T) {
		builderWithJunior := builder.AddJunior(11111)
		member, err := builderWithJunior.Build()
		if err != nil {
			t.Fatalf("Failed to build with junior: %v", err)
		}

		if len(member.JuniorIds()) != 1 {
			t.Errorf("Expected 1 junior, got %d", len(member.JuniorIds()))
		}

		if member.JuniorIds()[0] != 11111 {
			t.Errorf("Expected junior ID %d, got %d", 11111, member.JuniorIds()[0])
		}
	})

	t.Run("AddMultipleJuniors", func(t *testing.T) {
		// Create a fresh builder for this test
		freshBuilder := NewBuilder(characterId, tenantId, level, worldId)
		builderWithJuniors := freshBuilder.AddJunior(11111).AddJunior(22222)
		member, err := builderWithJuniors.Build()
		if err != nil {
			t.Fatalf("Failed to build with two juniors: %v", err)
		}

		if len(member.JuniorIds()) != 2 {
			t.Errorf("Expected 2 juniors, got %d", len(member.JuniorIds()))
		}
	})

	t.Run("AddTooManyJuniors", func(t *testing.T) {
		freshBuilder := NewBuilder(characterId, tenantId, level, worldId)
		builderWithTooManyJuniors := freshBuilder.AddJunior(11111).AddJunior(22222).AddJunior(33333)
		_, err := builderWithTooManyJuniors.Build()
		if err == nil {
			t.Error("Should fail when adding more than 2 juniors")
		}
	})

	t.Run("RemoveJunior", func(t *testing.T) {
		freshBuilder := NewBuilder(characterId, tenantId, level, worldId)
		builderWithJuniors := freshBuilder.AddJunior(11111).AddJunior(22222)
		builderRemovedJunior := builderWithJuniors.RemoveJunior(11111)
		member, err := builderRemovedJunior.Build()
		if err != nil {
			t.Fatalf("Failed to build after removing junior: %v", err)
		}

		if len(member.JuniorIds()) != 1 {
			t.Errorf("Expected 1 junior after removal, got %d", len(member.JuniorIds()))
		}

		if member.JuniorIds()[0] != 22222 {
			t.Errorf("Expected remaining junior ID %d, got %d", 22222, member.JuniorIds()[0])
		}
	})

	t.Run("AddRep", func(t *testing.T) {
		freshBuilder := NewBuilder(characterId, tenantId, level, worldId)
		builderWithRep := freshBuilder.SetRep(1000).AddRep(500)
		member, err := builderWithRep.Build()
		if err != nil {
			t.Fatalf("Failed to build with added rep: %v", err)
		}

		if member.Rep() != 1500 {
			t.Errorf("Expected Rep %d, got %d", 1500, member.Rep())
		}
	})

	t.Run("SubtractRep", func(t *testing.T) {
		freshBuilder := NewBuilder(characterId, tenantId, level, worldId)
		builderWithRep := freshBuilder.SetRep(1000).SubtractRep(300)
		member, err := builderWithRep.Build()
		if err != nil {
			t.Fatalf("Failed to build with subtracted rep: %v", err)
		}

		if member.Rep() != 700 {
			t.Errorf("Expected Rep %d, got %d", 700, member.Rep())
		}
	})

	t.Run("SubtractTooMuchRep", func(t *testing.T) {
		freshBuilder := NewBuilder(characterId, tenantId, level, worldId)
		builderWithRep := freshBuilder.SetRep(1000).SubtractRep(1500)
		member, err := builderWithRep.Build()
		if err != nil {
			t.Fatalf("Failed to build with subtracted rep: %v", err)
		}

		if member.Rep() != 0 {
			t.Errorf("Expected Rep %d (capped at 0), got %d", 0, member.Rep())
		}
	})

	t.Run("AddDailyRep", func(t *testing.T) {
		freshBuilder := NewBuilder(characterId, tenantId, level, worldId)
		builderWithDailyRep := freshBuilder.SetDailyRep(100).AddDailyRep(200)
		member, err := builderWithDailyRep.Build()
		if err != nil {
			t.Fatalf("Failed to build with added daily rep: %v", err)
		}

		if member.DailyRep() != 300 {
			t.Errorf("Expected DailyRep %d, got %d", 300, member.DailyRep())
		}
	})

	t.Run("ResetDailyRep", func(t *testing.T) {
		freshBuilder := NewBuilder(characterId, tenantId, level, worldId)
		builderWithReset := freshBuilder.SetDailyRep(1000).ResetDailyRep()
		member, err := builderWithReset.Build()
		if err != nil {
			t.Fatalf("Failed to build with reset daily rep: %v", err)
		}

		if member.DailyRep() != 0 {
			t.Errorf("Expected DailyRep %d, got %d", 0, member.DailyRep())
		}
	})

	t.Run("ClearSenior", func(t *testing.T) {
		freshBuilder := NewBuilder(characterId, tenantId, level, worldId)
		builderWithoutSenior := freshBuilder.SetSeniorId(99999).ClearSeniorId()
		member, err := builderWithoutSenior.Build()
		if err != nil {
			t.Fatalf("Failed to build with cleared senior: %v", err)
		}

		if member.SeniorId() != nil {
			t.Errorf("Expected SeniorId to be nil, got %d", *member.SeniorId())
		}
	})

	t.Run("ClearJuniors", func(t *testing.T) {
		freshBuilder := NewBuilder(characterId, tenantId, level, worldId)
		builderWithoutJuniors := freshBuilder.AddJunior(11111).AddJunior(22222).SetJuniorIds([]uint32{})
		member, err := builderWithoutJuniors.Build()
		if err != nil {
			t.Fatalf("Failed to build with cleared juniors: %v", err)
		}

		if len(member.JuniorIds()) != 0 {
			t.Errorf("Expected 0 juniors, got %d", len(member.JuniorIds()))
		}
	})
}

func TestBuilder_ModificationWorkflow(t *testing.T) {
	characterId := uint32(12345)
	tenantId := uuid.New()
	level := uint16(50)
	worldId := world.Id(1)

	// Create initial member
	original, err := NewBuilder(characterId, tenantId, level, worldId).
		SetSeniorId(99999).
		SetJuniorIds([]uint32{11111}).
		SetRep(1000).
		SetDailyRep(100).
		Build()

	if err != nil {
		t.Fatalf("Failed to create original member: %v", err)
	}

	// Test modification workflow using Builder() method
	modified, err := original.Builder().
		AddJunior(22222).
		AddRep(500).
		AddDailyRep(50).
		Build()

	if err != nil {
		t.Fatalf("Failed to modify member: %v", err)
	}

	// Verify original is unchanged
	if len(original.JuniorIds()) != 1 {
		t.Errorf("Original should have 1 junior, got %d", len(original.JuniorIds()))
	}

	if original.Rep() != 1000 {
		t.Errorf("Original rep should be 1000, got %d", original.Rep())
	}

	if original.DailyRep() != 100 {
		t.Errorf("Original daily rep should be 100, got %d", original.DailyRep())
	}

	// Verify modified member has changes
	if len(modified.JuniorIds()) != 2 {
		t.Errorf("Modified should have 2 juniors, got %d", len(modified.JuniorIds()))
	}

	if modified.Rep() != 1500 {
		t.Errorf("Modified rep should be 1500, got %d", modified.Rep())
	}

	if modified.DailyRep() != 150 {
		t.Errorf("Modified daily rep should be 150, got %d", modified.DailyRep())
	}

	// Verify base fields are preserved
	if modified.CharacterId() != original.CharacterId() {
		t.Error("CharacterId should be preserved in modification")
	}

	if modified.SeniorId() != original.SeniorId() {
		t.Error("SeniorId should be preserved in modification")
	}
}

func TestBuilder_ValidationErrors(t *testing.T) {
	characterId := uint32(12345)
	tenantId := uuid.New()
	level := uint16(50)
	worldId := world.Id(1)

	tests := []struct {
		name        string
		setupBuilder func() *Builder
		expectError bool
	}{
		{
			name: "Valid builder",
			setupBuilder: func() *Builder {
				return NewBuilder(characterId, tenantId, level, worldId)
			},
			expectError: false,
		},
		{
			name: "Too many juniors",
			setupBuilder: func() *Builder {
				return NewBuilder(characterId, tenantId, level, worldId).
					AddJunior(11111).
					AddJunior(22222).
					AddJunior(33333)
			},
			expectError: true,
		},
		{
			name: "Daily rep over cap",
			setupBuilder: func() *Builder {
				return NewBuilder(characterId, tenantId, level, worldId).
					SetDailyRep(6000)
			},
			expectError: true,
		},
		{
			name: "Self reference as senior",
			setupBuilder: func() *Builder {
				return NewBuilder(characterId, tenantId, level, worldId).
					SetSeniorId(characterId)
			},
			expectError: true,
		},
		{
			name: "Self reference in juniors",
			setupBuilder: func() *Builder {
				return NewBuilder(characterId, tenantId, level, worldId).
					AddJunior(characterId)
			},
			expectError: true,
		},
		{
			name: "Duplicate juniors",
			setupBuilder: func() *Builder {
				return NewBuilder(characterId, tenantId, level, worldId).
					AddJunior(11111).
					AddJunior(11111)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := tt.setupBuilder()
			_, err := builder.Build()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestBuilder_Immutability(t *testing.T) {
	characterId := uint32(12345)
	tenantId := uuid.New()
	level := uint16(50)
	worldId := world.Id(1)

	// Create two independent builders 
	builder1 := NewBuilder(characterId, tenantId, level, worldId).
		SetSeniorId(99999).
		SetRep(1000).
		AddJunior(11111).
		SetDailyRep(100)
	
	builder2 := NewBuilder(characterId, tenantId, level, worldId).
		SetSeniorId(99999).
		SetRep(1000).
		AddJunior(22222).
		SetDailyRep(200)

	// Build both members
	member1, err1 := builder1.Build()
	member2, err2 := builder2.Build()

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to build members: %v, %v", err1, err2)
	}

	// Verify they have different juniors
	if len(member1.JuniorIds()) != 1 || member1.JuniorIds()[0] != 11111 {
		t.Error("Member1 should have junior 11111")
	}

	if len(member2.JuniorIds()) != 1 || member2.JuniorIds()[0] != 22222 {
		t.Error("Member2 should have junior 22222")
	}

	// Verify they have different daily rep
	if member1.DailyRep() != 100 {
		t.Errorf("Member1 daily rep should be 100, got %d", member1.DailyRep())
	}

	if member2.DailyRep() != 200 {
		t.Errorf("Member2 daily rep should be 200, got %d", member2.DailyRep())
	}

	// Verify they share the same base values
	if member1.SeniorId() == nil || member2.SeniorId() == nil || *member1.SeniorId() != *member2.SeniorId() {
		t.Error("Both members should have same senior ID")
	}

	if member1.Rep() != member2.Rep() {
		t.Error("Both members should have same rep")
	}
}