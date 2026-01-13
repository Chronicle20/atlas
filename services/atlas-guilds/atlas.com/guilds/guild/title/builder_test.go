package title

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilderValidation(t *testing.T) {
	validTenantId := uuid.New()
	validId := uuid.New()
	validGuildId := uint32(1)
	validName := "Guild Master"
	validIndex := byte(0)

	tests := []struct {
		name    string
		setup   func() *Builder
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid builder succeeds",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validId, validGuildId, validName, validIndex)
			},
			wantErr: false,
		},
		{
			name: "zero guild id fails",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validId, 0, validName, validIndex)
			},
			wantErr: true,
			errMsg:  "guild ID must be greater than 0",
		},
		{
			name: "empty name fails",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validId, validGuildId, "", validIndex)
			},
			wantErr: true,
			errMsg:  "title name is required",
		},
		{
			name: "index zero is valid",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validId, validGuildId, validName, 0)
			},
			wantErr: false,
		},
		{
			name: "high index is valid",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validId, validGuildId, validName, 255)
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

func TestModelBuilder(t *testing.T) {
	tenantId := uuid.New()
	titleId := uuid.New()

	original, err := NewBuilder(tenantId, titleId, 1, "Original Name", 0).Build()
	require.NoError(t, err)

	modified, err := original.Builder().
		SetName("Modified Name").
		Build()
	require.NoError(t, err)

	assert.Equal(t, original.id, modified.id)
	assert.Equal(t, original.guildId, modified.guildId)
	assert.Equal(t, original.index, modified.index)
	assert.NotEqual(t, original.name, modified.name)
}

func TestBuilderFieldValues(t *testing.T) {
	tenantId := uuid.New()
	titleId := uuid.New()
	guildId := uint32(5)
	name := "Junior Member"
	index := byte(4)

	model, err := NewBuilder(tenantId, titleId, guildId, name, index).Build()
	require.NoError(t, err)

	assert.Equal(t, tenantId, model.tenantId)
	assert.Equal(t, titleId, model.id)
	assert.Equal(t, guildId, model.guildId)
	assert.Equal(t, name, model.name)
	assert.Equal(t, index, model.index)
}

func TestModelBuilder_DoesNotMutateOriginal(t *testing.T) {
	tenantId := uuid.New()
	titleId := uuid.New()
	original, err := NewBuilder(tenantId, titleId, 1, "Guild Master", 0).Build()
	require.NoError(t, err)

	// Capture original values
	originalName := original.name
	originalIndex := original.index

	// Get builder from model and modify it
	builder := original.Builder()
	builder.SetName("Modified Name")
	builder.SetIndex(2)

	// Build new model
	modified, err := builder.Build()
	require.NoError(t, err)

	// Verify original model is unchanged
	assert.Equal(t, originalName, original.name, "original name should not be mutated")
	assert.Equal(t, originalIndex, original.index, "original index should not be mutated")

	// Verify modified model has new values
	assert.Equal(t, "Modified Name", modified.name)
	assert.Equal(t, byte(2), modified.index)
}
