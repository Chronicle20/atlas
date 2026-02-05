package guild

import (
	"testing"

	world2 "github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilderValidation(t *testing.T) {
	validTenantId := uuid.New()
	validId := uint32(1)
	validWorldId := world2.Id(0)
	validName := "TestGuild"
	validLeaderId := uint32(100)

	tests := []struct {
		name    string
		setup   func() *Builder
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid builder succeeds",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validId, validWorldId, validName, validLeaderId)
			},
			wantErr: false,
		},
		{
			name: "zero guild id fails",
			setup: func() *Builder {
				return NewBuilder(validTenantId, 0, validWorldId, validName, validLeaderId)
			},
			wantErr: true,
			errMsg:  "guild ID must be greater than 0",
		},
		{
			name: "empty name fails",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validId, validWorldId, "", validLeaderId)
			},
			wantErr: true,
			errMsg:  "guild name is required",
		},
		{
			name: "zero leader id fails",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validId, validWorldId, validName, 0)
			},
			wantErr: true,
			errMsg:  "leader ID must be greater than 0",
		},
		{
			name: "zero capacity fails",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validId, validWorldId, validName, validLeaderId).
					SetCapacity(0)
			},
			wantErr: true,
			errMsg:  "capacity must be greater than 0",
		},
		{
			name: "valid with all optional fields",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validId, validWorldId, validName, validLeaderId).
					SetNotice("Test notice").
					SetPoints(100).
					SetCapacity(50).
					SetLogo(1).
					SetLogoColor(2).
					SetLogoBackground(3).
					SetLogoBackgroundColor(4)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := tt.setup()
			model, err := builder.Build()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, model)
			}
		})
	}
}

func TestBuilderDefaults(t *testing.T) {
	tenantId := uuid.New()
	builder := NewBuilder(tenantId, 1, 0, "TestGuild", 100)

	model, err := builder.Build()
	require.NoError(t, err)

	assert.Equal(t, uint32(30), model.Capacity(), "default capacity should be 30")
}

func TestModelBuilder(t *testing.T) {
	tenantId := uuid.New()
	original, err := NewBuilder(tenantId, 1, 0, "TestGuild", 100).
		SetNotice("Original notice").
		SetCapacity(50).
		Build()
	require.NoError(t, err)

	modified, err := original.Builder().
		SetNotice("Modified notice").
		Build()
	require.NoError(t, err)

	assert.Equal(t, original.Id(), modified.Id())
	assert.Equal(t, original.Capacity(), modified.Capacity())
}

func TestBuilderImmutability(t *testing.T) {
	tenantId := uuid.New()
	builder := NewBuilder(tenantId, 1, 0, "TestGuild", 100)

	model1, err := builder.Build()
	require.NoError(t, err)

	builder.SetNotice("Changed")
	model2, err := builder.Build()
	require.NoError(t, err)

	assert.NotEqual(t, model1, model2, "builder modifications should create different models")
}

func TestModelBuilder_DoesNotMutateOriginal(t *testing.T) {
	tenantId := uuid.New()
	original, err := NewBuilder(tenantId, 1, 0, "TestGuild", 100).
		SetNotice("Original notice").
		SetCapacity(50).
		SetPoints(100).
		Build()
	require.NoError(t, err)

	// Capture original values (using private fields since we're in same package)
	originalNotice := original.notice
	originalCapacity := original.capacity
	originalPoints := original.points

	// Get builder from model and modify it
	builder := original.Builder()
	builder.SetNotice("Modified notice")
	builder.SetCapacity(75)
	builder.SetPoints(200)

	// Build new model
	modified, err := builder.Build()
	require.NoError(t, err)

	// Verify original model is unchanged
	assert.Equal(t, originalNotice, original.notice, "original notice should not be mutated")
	assert.Equal(t, originalCapacity, original.capacity, "original capacity should not be mutated")
	assert.Equal(t, originalPoints, original.points, "original points should not be mutated")

	// Verify modified model has new values
	assert.Equal(t, "Modified notice", modified.notice)
	assert.Equal(t, uint32(75), modified.capacity)
	assert.Equal(t, uint32(200), modified.points)
}
