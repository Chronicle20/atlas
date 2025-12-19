package party

import (
	"github.com/google/uuid"
	"testing"
)

func TestBuilder_Build_Success(t *testing.T) {
	tenantId := uuid.New()
	partyId := uint32(123)
	leaderId := uint32(456)
	members := []uint32{456, 789}

	builder := NewBuilder(tenantId, partyId, leaderId)
	builder.SetMembers(members)

	model, err := builder.Build()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if model.TenantId() != tenantId {
		t.Errorf("Expected tenant ID %v, got %v", tenantId, model.TenantId())
	}
	if model.Id() != partyId {
		t.Errorf("Expected party ID %d, got %d", partyId, model.Id())
	}
	if model.LeaderId() != leaderId {
		t.Errorf("Expected leader ID %d, got %d", leaderId, model.LeaderId())
	}
	if len(model.Members()) != len(members) {
		t.Errorf("Expected %d members, got %d", len(members), len(model.Members()))
	}
}

func TestBuilder_Build_ValidationErrors(t *testing.T) {
	tenantId := uuid.New()
	partyId := uint32(123)
	leaderId := uint32(456)

	tests := []struct {
		name        string
		setupFn     func() *Builder
		expectedErr string
	}{
		{
			name: "zero party ID",
			setupFn: func() *Builder {
				return NewBuilder(tenantId, 0, leaderId)
			},
			expectedErr: "party ID must be greater than 0",
		},
		{
			name: "zero leader ID",
			setupFn: func() *Builder {
				return NewBuilder(tenantId, partyId, 0)
			},
			expectedErr: "leader ID must be greater than 0",
		},
		{
			name: "leader not in members",
			setupFn: func() *Builder {
				builder := NewBuilder(tenantId, partyId, leaderId)
				builder.SetMembers([]uint32{789, 101})
				return builder
			},
			expectedErr: "leader must be a member of the party",
		},
		{
			name: "duplicate members",
			setupFn: func() *Builder {
				builder := NewBuilder(tenantId, partyId, leaderId)
				builder.SetMembers([]uint32{456, 789, 456})
				return builder
			},
			expectedErr: "duplicate member ID found",
		},
		{
			name: "zero member ID",
			setupFn: func() *Builder {
				builder := NewBuilder(tenantId, partyId, leaderId)
				builder.SetMembers([]uint32{456, 0})
				return builder
			},
			expectedErr: "member ID must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := tt.setupFn()
			_, err := builder.Build()
			if err == nil {
				t.Fatalf("Expected error, got none")
			}
			if err.Error() != tt.expectedErr {
				t.Errorf("Expected error '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}

func TestModel_Builder(t *testing.T) {
	tenantId := uuid.New()
	partyId := uint32(123)
	leaderId := uint32(456)
	members := []uint32{456, 789}

	// Create original model
	original := Model{
		tenantId: tenantId,
		id:       partyId,
		leaderId: leaderId,
		members:  members,
	}

	// Create builder from model
	builder := original.Builder()

	// Modify and rebuild
	newMembers := []uint32{456, 789, 101}
	builder.SetMembers(newMembers)

	modified, err := builder.Build()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify modification
	if len(modified.Members()) != len(newMembers) {
		t.Errorf("Expected %d members, got %d", len(newMembers), len(modified.Members()))
	}

	// Verify original is unchanged
	if len(original.Members()) != len(members) {
		t.Errorf("Original model was modified, expected %d members, got %d", len(members), len(original.Members()))
	}
}

func TestBuilder_FluentInterface(t *testing.T) {
	tenantId := uuid.New()
	partyId := uint32(123)
	leaderId := uint32(456)
	members := []uint32{456, 789}

	// Test fluent interface
	model, err := NewBuilder(tenantId, partyId, leaderId).
		SetMembers(members).
		Build()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if model.Id() != partyId {
		t.Errorf("Expected party ID %d, got %d", partyId, model.Id())
	}
}