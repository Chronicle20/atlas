package thread

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilderValidation(t *testing.T) {
	validTenantId := uuid.New()
	validGuildId := uint32(1)
	validId := uint32(1)
	validPosterId := uint32(100)
	validTitle := "Test Thread"
	validMessage := "Test message content"

	tests := []struct {
		name    string
		setup   func() *Builder
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid builder succeeds",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validGuildId, validId, validPosterId, validTitle, validMessage)
			},
			wantErr: false,
		},
		{
			name: "zero guild id fails",
			setup: func() *Builder {
				return NewBuilder(validTenantId, 0, validId, validPosterId, validTitle, validMessage)
			},
			wantErr: true,
			errMsg:  "guild ID must be greater than 0",
		},
		{
			name: "zero thread id fails",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validGuildId, 0, validPosterId, validTitle, validMessage)
			},
			wantErr: true,
			errMsg:  "thread ID must be greater than 0",
		},
		{
			name: "zero poster id fails",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validGuildId, validId, 0, validTitle, validMessage)
			},
			wantErr: true,
			errMsg:  "poster ID must be greater than 0",
		},
		{
			name: "empty title fails",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validGuildId, validId, validPosterId, "", validMessage)
			},
			wantErr: true,
			errMsg:  "thread title is required",
		},
		{
			name: "valid with all optional fields",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validGuildId, validId, validPosterId, validTitle, validMessage).
					SetEmoticonId(5).
					SetNotice(true).
					SetCreatedAt(time.Now())
			},
			wantErr: false,
		},
		{
			name: "empty message is allowed",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validGuildId, validId, validPosterId, validTitle, "")
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
	before := time.Now()

	builder := NewBuilder(tenantId, 1, 1, 100, "Test", "Message")
	model, err := builder.Build()
	require.NoError(t, err)

	after := time.Now()

	assert.False(t, model.createdAt.Before(before), "createdAt should be set to now or after")
	assert.False(t, model.createdAt.After(after), "createdAt should be set to now or before")
	assert.False(t, model.notice, "default notice should be false")
	assert.Equal(t, uint32(0), model.emoticonId, "default emoticonId should be 0")
}

func TestModelBuilder(t *testing.T) {
	tenantId := uuid.New()
	createdAt := time.Now().Add(-time.Hour)

	original, err := NewBuilder(tenantId, 1, 1, 100, "Original Title", "Original Message").
		SetNotice(true).
		SetCreatedAt(createdAt).
		Build()
	require.NoError(t, err)

	modified, err := original.Builder().
		SetNotice(false).
		Build()
	require.NoError(t, err)

	assert.Equal(t, original.Id(), modified.Id())
	assert.Equal(t, original.title, modified.title)
	assert.Equal(t, original.createdAt, modified.createdAt)
	assert.NotEqual(t, original.notice, modified.notice)
}

func TestModelBuilder_DoesNotMutateOriginal(t *testing.T) {
	tenantId := uuid.New()
	createdAt := time.Now().Add(-time.Hour)

	original, err := NewBuilder(tenantId, 1, 1, 100, "Original Title", "Original Message").
		SetNotice(true).
		SetEmoticonId(5).
		SetCreatedAt(createdAt).
		Build()
	require.NoError(t, err)

	// Capture original values
	originalTitle := original.title
	originalMessage := original.message
	originalNotice := original.notice
	originalEmoticonId := original.emoticonId

	// Get builder from model and modify it
	builder := original.Builder()
	builder.SetNotice(false)
	builder.SetEmoticonId(10)

	// Build new model
	modified, err := builder.Build()
	require.NoError(t, err)

	// Verify original model is unchanged
	assert.Equal(t, originalTitle, original.title, "original title should not be mutated")
	assert.Equal(t, originalMessage, original.message, "original message should not be mutated")
	assert.Equal(t, originalNotice, original.notice, "original notice should not be mutated")
	assert.Equal(t, originalEmoticonId, original.emoticonId, "original emoticonId should not be mutated")

	// Verify modified model has new values
	assert.False(t, modified.notice)
	assert.Equal(t, uint32(10), modified.emoticonId)
}
