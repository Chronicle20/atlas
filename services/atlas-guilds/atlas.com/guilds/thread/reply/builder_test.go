package reply

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilderValidation(t *testing.T) {
	validId := uint32(1)
	validPosterId := uint32(100)
	validMessage := "Test reply message"

	tests := []struct {
		name    string
		setup   func() *Builder
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid builder succeeds",
			setup: func() *Builder {
				return NewBuilder(validId, validPosterId, validMessage)
			},
			wantErr: false,
		},
		{
			name: "zero reply id fails",
			setup: func() *Builder {
				return NewBuilder(0, validPosterId, validMessage)
			},
			wantErr: true,
			errMsg:  "reply ID must be greater than 0",
		},
		{
			name: "zero poster id fails",
			setup: func() *Builder {
				return NewBuilder(validId, 0, validMessage)
			},
			wantErr: true,
			errMsg:  "poster ID must be greater than 0",
		},
		{
			name: "valid with custom created at",
			setup: func() *Builder {
				return NewBuilder(validId, validPosterId, validMessage).
					SetCreatedAt(time.Now().Add(-time.Hour))
			},
			wantErr: false,
		},
		{
			name: "empty message is allowed",
			setup: func() *Builder {
				return NewBuilder(validId, validPosterId, "")
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
	before := time.Now()

	builder := NewBuilder(1, 100, "Test message")
	model, err := builder.Build()
	require.NoError(t, err)

	after := time.Now()

	assert.False(t, model.createdAt.Before(before), "createdAt should be set to now or after")
	assert.False(t, model.createdAt.After(after), "createdAt should be set to now or before")
}

func TestModelBuilder(t *testing.T) {
	customTime := time.Now().Add(-24 * time.Hour)

	original, err := NewBuilder(1, 100, "Original message").
		SetCreatedAt(customTime).
		Build()
	require.NoError(t, err)

	modified, err := original.Builder().Build()
	require.NoError(t, err)

	assert.Equal(t, original.id, modified.id)
	assert.Equal(t, original.posterId, modified.posterId)
	assert.Equal(t, original.message, modified.message)
	assert.Equal(t, original.createdAt, modified.createdAt)
}

func TestBuilderFieldValues(t *testing.T) {
	id := uint32(5)
	posterId := uint32(200)
	message := "Specific reply content"
	createdAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	model, err := NewBuilder(id, posterId, message).
		SetCreatedAt(createdAt).
		Build()
	require.NoError(t, err)

	assert.Equal(t, id, model.id)
	assert.Equal(t, posterId, model.posterId)
	assert.Equal(t, message, model.message)
	assert.Equal(t, createdAt, model.createdAt)
}

func TestModelBuilder_DoesNotMutateOriginal(t *testing.T) {
	originalTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	modifiedTime := time.Date(2024, 6, 20, 15, 45, 0, 0, time.UTC)

	original, err := NewBuilder(1, 100, "Original message").
		SetCreatedAt(originalTime).
		Build()
	require.NoError(t, err)

	// Capture original values
	originalMessage := original.message
	originalCreatedAt := original.createdAt

	// Get builder from model and modify it
	builder := original.Builder()
	builder.SetCreatedAt(modifiedTime)

	// Build new model
	modified, err := builder.Build()
	require.NoError(t, err)

	// Verify original model is unchanged
	assert.Equal(t, originalMessage, original.message, "original message should not be mutated")
	assert.Equal(t, originalCreatedAt, original.createdAt, "original createdAt should not be mutated")

	// Verify modified model has new values
	assert.Equal(t, modifiedTime, modified.createdAt)
}
