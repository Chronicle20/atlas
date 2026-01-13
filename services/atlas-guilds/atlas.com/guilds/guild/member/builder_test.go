package member

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilderValidation(t *testing.T) {
	validTenantId := uuid.New()
	validGuildId := uint32(1)
	validCharacterId := uint32(100)
	validName := "TestMember"

	tests := []struct {
		name    string
		setup   func() *Builder
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid builder succeeds",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validGuildId, validCharacterId, validName)
			},
			wantErr: false,
		},
		{
			name: "zero guild id fails",
			setup: func() *Builder {
				return NewBuilder(validTenantId, 0, validCharacterId, validName)
			},
			wantErr: true,
			errMsg:  "guild ID must be greater than 0",
		},
		{
			name: "zero character id fails",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validGuildId, 0, validName)
			},
			wantErr: true,
			errMsg:  "character ID must be greater than 0",
		},
		{
			name: "empty name fails",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validGuildId, validCharacterId, "")
			},
			wantErr: true,
			errMsg:  "member name is required",
		},
		{
			name: "valid with all optional fields",
			setup: func() *Builder {
				return NewBuilder(validTenantId, validGuildId, validCharacterId, validName).
					SetJobId(100).
					SetLevel(50).
					SetTitle(1).
					SetOnline(true).
					SetAllianceTitle(2)
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
	builder := NewBuilder(tenantId, 1, 100, "TestMember")

	model, err := builder.Build()
	require.NoError(t, err)

	assert.Equal(t, uint16(0), model.jobId, "default jobId should be 0")
	assert.Equal(t, byte(0), model.level, "default level should be 0")
	assert.Equal(t, byte(0), model.title, "default title should be 0")
	assert.False(t, model.online, "default online should be false")
	assert.Equal(t, byte(0), model.allianceTitle, "default allianceTitle should be 0")
}

func TestModelBuilder(t *testing.T) {
	tenantId := uuid.New()
	original, err := NewBuilder(tenantId, 1, 100, "TestMember").
		SetJobId(100).
		SetLevel(50).
		SetOnline(true).
		Build()
	require.NoError(t, err)

	modified, err := original.Builder().
		SetOnline(false).
		Build()
	require.NoError(t, err)

	assert.Equal(t, original.characterId, modified.characterId)
	assert.Equal(t, original.jobId, modified.jobId)
	assert.Equal(t, original.level, modified.level)
	assert.NotEqual(t, original.online, modified.online)
}

func TestBuilderFieldValues(t *testing.T) {
	tenantId := uuid.New()
	guildId := uint32(5)
	characterId := uint32(200)
	name := "SpecificName"
	jobId := uint16(300)
	level := byte(99)
	title := byte(1)
	allianceTitle := byte(2)

	model, err := NewBuilder(tenantId, guildId, characterId, name).
		SetJobId(jobId).
		SetLevel(level).
		SetTitle(title).
		SetOnline(true).
		SetAllianceTitle(allianceTitle).
		Build()
	require.NoError(t, err)

	assert.Equal(t, tenantId, model.tenantId)
	assert.Equal(t, guildId, model.guildId)
	assert.Equal(t, characterId, model.characterId)
	assert.Equal(t, name, model.name)
	assert.Equal(t, jobId, model.jobId)
	assert.Equal(t, level, model.level)
	assert.Equal(t, title, model.title)
	assert.True(t, model.online)
	assert.Equal(t, allianceTitle, model.allianceTitle)
}

func TestModelBuilder_DoesNotMutateOriginal(t *testing.T) {
	tenantId := uuid.New()
	original, err := NewBuilder(tenantId, 1, 100, "TestMember").
		SetJobId(100).
		SetLevel(50).
		SetTitle(1).
		SetOnline(true).
		Build()
	require.NoError(t, err)

	// Capture original values
	originalJobId := original.jobId
	originalLevel := original.level
	originalTitle := original.title
	originalOnline := original.online

	// Get builder from model and modify it
	builder := original.Builder()
	builder.SetJobId(200)
	builder.SetLevel(75)
	builder.SetTitle(2)
	builder.SetOnline(false)

	// Build new model
	modified, err := builder.Build()
	require.NoError(t, err)

	// Verify original model is unchanged
	assert.Equal(t, originalJobId, original.jobId, "original jobId should not be mutated")
	assert.Equal(t, originalLevel, original.level, "original level should not be mutated")
	assert.Equal(t, originalTitle, original.title, "original title should not be mutated")
	assert.Equal(t, originalOnline, original.online, "original online should not be mutated")

	// Verify modified model has new values
	assert.Equal(t, uint16(200), modified.jobId)
	assert.Equal(t, byte(75), modified.level)
	assert.Equal(t, byte(2), modified.title)
	assert.False(t, modified.online)
}
