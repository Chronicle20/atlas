package reactor

import (
	"atlas-reactors/reactor/data"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestNewModelBuilder tests the builder constructor
func TestNewModelBuilder(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	builder := NewModelBuilder(ten, 1, 2, 100000, 2000000, "test-reactor")

	assert.NotNil(t, builder)
	assert.Equal(t, uint32(2000000), builder.Classification())
}

// TestModelBuilder_SetMethods tests all setter methods
func TestModelBuilder_SetMethods(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	builder := NewModelBuilder(ten, 1, 2, 100000, 2000000, "test-reactor")

	// Test SetState
	result := builder.SetState(5)
	assert.Same(t, builder, result, "SetState should return same builder for chaining")

	// Test SetPosition
	result = builder.SetPosition(150, 250)
	assert.Same(t, builder, result, "SetPosition should return same builder for chaining")

	// Test SetDelay
	result = builder.SetDelay(1000)
	assert.Same(t, builder, result, "SetDelay should return same builder for chaining")

	// Test SetDirection
	result = builder.SetDirection(4)
	assert.Same(t, builder, result, "SetDirection should return same builder for chaining")

	// Test SetData
	testData := data.Model{}
	result = builder.SetData(testData)
	assert.Same(t, builder, result, "SetData should return same builder for chaining")

	// Test SetId
	result = builder.SetId(12345)
	assert.Same(t, builder, result, "SetId should return same builder for chaining")

	// Test UpdateTime
	result = builder.UpdateTime()
	assert.Same(t, builder, result, "UpdateTime should return same builder for chaining")
}

// TestModelBuilder_Build tests that Build produces correct Model
func TestModelBuilder_Build(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	testData := data.Model{}

	builder := NewModelBuilder(ten, 1, 2, 100000, 2000000, "test-reactor").
		SetState(3).
		SetPosition(150, 250).
		SetDelay(500).
		SetDirection(2).
		SetData(testData).
		SetId(12345)

	model, err := builder.Build()

	assert.NoError(t, err)
	assert.Equal(t, ten, model.Tenant())
	assert.Equal(t, uint32(12345), model.Id())
	assert.Equal(t, byte(1), model.WorldId())
	assert.Equal(t, byte(2), model.ChannelId())
	assert.Equal(t, uint32(100000), model.MapId())
	assert.Equal(t, uint32(2000000), model.Classification())
	assert.Equal(t, "test-reactor", model.Name())
	assert.Equal(t, int8(3), model.State())
	assert.Equal(t, int16(150), model.X())
	assert.Equal(t, int16(250), model.Y())
	assert.Equal(t, uint32(500), model.Delay())
	assert.Equal(t, byte(2), model.Direction())
}

// TestModelBuilder_Build_DefaultValues tests Build with minimal configuration
func TestModelBuilder_Build_DefaultValues(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	builder := NewModelBuilder(ten, 1, 2, 100000, 2000000, "test-reactor")
	model, err := builder.Build()

	assert.NoError(t, err)

	// Required fields should be set
	assert.Equal(t, ten, model.Tenant())
	assert.Equal(t, byte(1), model.WorldId())
	assert.Equal(t, byte(2), model.ChannelId())
	assert.Equal(t, uint32(100000), model.MapId())
	assert.Equal(t, uint32(2000000), model.Classification())
	assert.Equal(t, "test-reactor", model.Name())

	// Optional fields should have zero values
	assert.Equal(t, uint32(0), model.Id())
	assert.Equal(t, int8(0), model.State())
	assert.Equal(t, int16(0), model.X())
	assert.Equal(t, int16(0), model.Y())
	assert.Equal(t, uint32(0), model.Delay())
	assert.Equal(t, byte(0), model.Direction())
	assert.Equal(t, byte(0), model.EventState())

	// UpdateTime should be set by constructor
	assert.False(t, model.UpdateTime().IsZero())
}

// TestModelBuilder_Build_ValidationErrors tests validation in Build
func TestModelBuilder_Build_ValidationErrors(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	tests := []struct {
		name    string
		builder *ModelBuilder
		wantErr string
	}{
		{
			name:    "missing classification",
			builder: NewModelBuilder(ten, 1, 2, 100000, 0, "test-reactor"),
			wantErr: "classification is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.builder.Build()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestNewFromModel tests creating a builder from existing model
func TestNewFromModel(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	testData := data.Model{}

	original, err := NewModelBuilder(ten, 1, 2, 100000, 2000000, "original-reactor").
		SetState(3).
		SetPosition(150, 250).
		SetDelay(500).
		SetDirection(2).
		SetData(testData).
		SetId(12345).
		Build()

	assert.NoError(t, err)

	builder := NewFromModel(original)

	assert.NotNil(t, builder)
	assert.Equal(t, original.Classification(), builder.Classification())
}

// TestNewFromModel_RoundTrip tests Model -> Builder -> Model consistency
func TestNewFromModel_RoundTrip(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	testData := data.Model{}

	original, err := NewModelBuilder(ten, 1, 2, 100000, 2000000, "test-reactor").
		SetState(3).
		SetPosition(150, 250).
		SetDelay(500).
		SetDirection(2).
		SetData(testData).
		SetId(12345).
		Build()

	assert.NoError(t, err)

	// Round trip: Model -> Builder -> Model
	rebuilt, err := NewFromModel(original).Build()

	assert.NoError(t, err)
	assert.Equal(t, original.Tenant(), rebuilt.Tenant())
	assert.Equal(t, original.Id(), rebuilt.Id())
	assert.Equal(t, original.WorldId(), rebuilt.WorldId())
	assert.Equal(t, original.ChannelId(), rebuilt.ChannelId())
	assert.Equal(t, original.MapId(), rebuilt.MapId())
	assert.Equal(t, original.Classification(), rebuilt.Classification())
	assert.Equal(t, original.Name(), rebuilt.Name())
	assert.Equal(t, original.State(), rebuilt.State())
	assert.Equal(t, original.X(), rebuilt.X())
	assert.Equal(t, original.Y(), rebuilt.Y())
	assert.Equal(t, original.Delay(), rebuilt.Delay())
	assert.Equal(t, original.Direction(), rebuilt.Direction())
	assert.Equal(t, original.EventState(), rebuilt.EventState())
	assert.Equal(t, original.UpdateTime(), rebuilt.UpdateTime())
}

// TestNewFromModel_Modification tests modifying a model through builder
func TestNewFromModel_Modification(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	original, err := NewModelBuilder(ten, 1, 2, 100000, 2000000, "test-reactor").
		SetState(0).
		SetPosition(100, 200).
		SetId(12345).
		Build()

	assert.NoError(t, err)

	// Create builder from model and modify
	modified, err := NewFromModel(original).
		SetState(5).
		SetPosition(300, 400).
		Build()

	assert.NoError(t, err)

	// Original should be unchanged (immutability)
	assert.Equal(t, int8(0), original.State())
	assert.Equal(t, int16(100), original.X())
	assert.Equal(t, int16(200), original.Y())

	// Modified should have new values
	assert.Equal(t, int8(5), modified.State())
	assert.Equal(t, int16(300), modified.X())
	assert.Equal(t, int16(400), modified.Y())

	// Unchanged fields should be preserved
	assert.Equal(t, original.Id(), modified.Id())
	assert.Equal(t, original.Name(), modified.Name())
	assert.Equal(t, original.Classification(), modified.Classification())
}

// TestModelBuilder_FluentChaining tests the fluent API pattern
func TestModelBuilder_FluentChaining(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	// Build using fluent chaining
	model, err := NewModelBuilder(ten, 1, 2, 100000, 2000000, "chained-reactor").
		SetId(99999).
		SetState(7).
		SetPosition(500, 600).
		SetDelay(1500).
		SetDirection(3).
		UpdateTime().
		Build()

	assert.NoError(t, err)
	assert.Equal(t, uint32(99999), model.Id())
	assert.Equal(t, int8(7), model.State())
	assert.Equal(t, int16(500), model.X())
	assert.Equal(t, int16(600), model.Y())
	assert.Equal(t, uint32(1500), model.Delay())
	assert.Equal(t, byte(3), model.Direction())
}

// TestModelBuilder_UpdateTime tests the UpdateTime method
func TestModelBuilder_UpdateTime(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	before := time.Now()
	time.Sleep(1 * time.Millisecond) // Ensure time difference

	builder := NewModelBuilder(ten, 1, 2, 100000, 2000000, "test-reactor")
	builder.UpdateTime()
	model, err := builder.Build()

	assert.NoError(t, err)

	time.Sleep(1 * time.Millisecond)
	after := time.Now()

	assert.True(t, model.UpdateTime().After(before) || model.UpdateTime().Equal(before))
	assert.True(t, model.UpdateTime().Before(after) || model.UpdateTime().Equal(after))
}

// TestModelBuilder_Classification tests the Classification getter
func TestModelBuilder_Classification(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	builder := NewModelBuilder(ten, 1, 2, 100000, 2000123, "test-reactor")

	assert.Equal(t, uint32(2000123), builder.Classification())
}
