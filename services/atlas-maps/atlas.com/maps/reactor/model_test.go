package reactor

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
)

func TestModel_Accessors(t *testing.T) {
	m := Model{
		id:             1,
		f:              field.NewBuilder(2, 3, 100000000).Build(),
		classification: 1001,
		name:           "test-reactor",
		state:          1,
		eventState:     2,
		delay:          1000,
		direction:      1,
		x:              100,
		y:              200,
		updateTime:     time.Now(),
	}

	if m.Id() != 1 {
		t.Errorf("Expected Id 1, got %d", m.Id())
	}
	if m.WorldId() != 2 {
		t.Errorf("Expected WorldId 2, got %d", m.WorldId())
	}
	if m.ChannelId() != 3 {
		t.Errorf("Expected ChannelId 3, got %d", m.ChannelId())
	}
	if m.MapId() != 100000000 {
		t.Errorf("Expected MapId 100000000, got %d", m.MapId())
	}
	if m.Classification() != 1001 {
		t.Errorf("Expected Classification 1001, got %d", m.Classification())
	}
	if m.Name() != "test-reactor" {
		t.Errorf("Expected Name 'test-reactor', got '%s'", m.Name())
	}
	if m.State() != 1 {
		t.Errorf("Expected State 1, got %d", m.State())
	}
	if m.EventState() != 2 {
		t.Errorf("Expected EventState 2, got %d", m.EventState())
	}
	if m.Delay() != 1000 {
		t.Errorf("Expected Delay 1000, got %d", m.Delay())
	}
	if m.Direction() != 1 {
		t.Errorf("Expected Direction 1, got %d", m.Direction())
	}
	if m.X() != 100 {
		t.Errorf("Expected X 100, got %d", m.X())
	}
	if m.Y() != 200 {
		t.Errorf("Expected Y 200, got %d", m.Y())
	}
}

func TestNewModelBuilder(t *testing.T) {
	f := field.NewBuilder(1, 2, 100000000).Build()
	builder := NewModelBuilder(f, 1001, "test-reactor")

	if builder.f.WorldId() != 1 {
		t.Errorf("Expected worldId 1, got %d", builder.f.WorldId())
	}
	if builder.f.ChannelId() != 2 {
		t.Errorf("Expected channelId 2, got %d", builder.f.ChannelId())
	}
	if builder.f.MapId() != 100000000 {
		t.Errorf("Expected mapId 100000000, got %d", builder.f.MapId())
	}
	if builder.classification != 1001 {
		t.Errorf("Expected classification 1001, got %d", builder.classification)
	}
	if builder.name != "test-reactor" {
		t.Errorf("Expected name 'test-reactor', got '%s'", builder.name)
	}
	if builder.updateTime.IsZero() {
		t.Error("Expected updateTime to be set")
	}
}

func TestModelBuilder_Build_Success(t *testing.T) {
	f := field.NewBuilder(1, 2, 100000000).Build()
	builder := NewModelBuilder(f, 1001, "test-reactor")
	builder.SetId(123)
	builder.SetState(1)
	builder.SetPosition(100, 200)
	builder.SetDelay(1000)
	builder.SetDirection(1)
	builder.SetEventState(2)

	model, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	if model.Id() != 123 {
		t.Errorf("Expected Id 123, got %d", model.Id())
	}
	if model.WorldId() != 1 {
		t.Errorf("Expected WorldId 1, got %d", model.WorldId())
	}
	if model.ChannelId() != 2 {
		t.Errorf("Expected ChannelId 2, got %d", model.ChannelId())
	}
	if model.MapId() != 100000000 {
		t.Errorf("Expected MapId 100000000, got %d", model.MapId())
	}
	if model.Classification() != 1001 {
		t.Errorf("Expected Classification 1001, got %d", model.Classification())
	}
	if model.Name() != "test-reactor" {
		t.Errorf("Expected Name 'test-reactor', got '%s'", model.Name())
	}
	if model.State() != 1 {
		t.Errorf("Expected State 1, got %d", model.State())
	}
	if model.EventState() != 2 {
		t.Errorf("Expected EventState 2, got %d", model.EventState())
	}
	if model.X() != 100 {
		t.Errorf("Expected X 100, got %d", model.X())
	}
	if model.Y() != 200 {
		t.Errorf("Expected Y 200, got %d", model.Y())
	}
	if model.Delay() != 1000 {
		t.Errorf("Expected Delay 1000, got %d", model.Delay())
	}
	if model.Direction() != 1 {
		t.Errorf("Expected Direction 1, got %d", model.Direction())
	}
}

func TestModelBuilder_Build_EmptyName_Error(t *testing.T) {
	f := field.NewBuilder(1, 2, 100000000).Build()
	builder := NewModelBuilder(f, 1001, "")

	_, err := builder.Build()
	if err == nil {
		t.Error("Build() expected error for empty name, got nil")
	}
	if err.Error() != "reactor name cannot be empty" {
		t.Errorf("Expected 'reactor name cannot be empty' error, got '%s'", err.Error())
	}
}

func TestModelBuilder_Build_ZeroClassification_Error(t *testing.T) {
	f := field.NewBuilder(1, 2, 100000000).Build()
	builder := NewModelBuilder(f, 0, "test-reactor")

	_, err := builder.Build()
	if err == nil {
		t.Error("Build() expected error for zero classification, got nil")
	}
	if err.Error() != "reactor classification must be positive" {
		t.Errorf("Expected 'reactor classification must be positive' error, got '%s'", err.Error())
	}
}

func TestNewFromModel(t *testing.T) {
	originalTime := time.Now()
	original := Model{
		id:             123,
		f:              field.NewBuilder(1, 2, 100000000).Build(),
		classification: 1001,
		name:           "test-reactor",
		state:          1,
		eventState:     2,
		delay:          1000,
		direction:      1,
		x:              100,
		y:              200,
		updateTime:     originalTime,
	}

	builder := NewFromModel(original)

	if builder.id != original.Id() {
		t.Errorf("Expected id %d, got %d", original.Id(), builder.id)
	}
	if builder.f.WorldId() != original.WorldId() {
		t.Errorf("Expected worldId %d, got %d", original.WorldId(), builder.f.WorldId())
	}
	if builder.f.ChannelId() != original.ChannelId() {
		t.Errorf("Expected channelId %d, got %d", original.ChannelId(), builder.f.ChannelId())
	}
	if builder.f.MapId() != original.MapId() {
		t.Errorf("Expected mapId %d, got %d", original.MapId(), builder.f.MapId())
	}
	if builder.classification != original.Classification() {
		t.Errorf("Expected classification %d, got %d", original.Classification(), builder.classification)
	}
	if builder.name != original.Name() {
		t.Errorf("Expected name '%s', got '%s'", original.Name(), builder.name)
	}
	if builder.state != original.State() {
		t.Errorf("Expected state %d, got %d", original.State(), builder.state)
	}
	if builder.eventState != original.EventState() {
		t.Errorf("Expected eventState %d, got %d", original.EventState(), builder.eventState)
	}
	if builder.x != original.X() {
		t.Errorf("Expected x %d, got %d", original.X(), builder.x)
	}
	if builder.y != original.Y() {
		t.Errorf("Expected y %d, got %d", original.Y(), builder.y)
	}
	if builder.delay != original.Delay() {
		t.Errorf("Expected delay %d, got %d", original.Delay(), builder.delay)
	}
	if builder.direction != original.Direction() {
		t.Errorf("Expected direction %d, got %d", original.Direction(), builder.direction)
	}
	if !builder.updateTime.Equal(originalTime) {
		t.Errorf("Expected updateTime %v, got %v", originalTime, builder.updateTime)
	}
}

func TestModelBuilder_SetMethods(t *testing.T) {
	f := field.NewBuilder(1, 2, 100000000).Build()
	builder := NewModelBuilder(f, 1001, "test-reactor")

	// Test chaining
	result := builder.SetId(123).SetState(1).SetPosition(100, 200).SetDelay(1000).SetDirection(1).SetEventState(2)

	if result != builder {
		t.Error("Set methods should return the builder for chaining")
	}

	if builder.id != 123 {
		t.Errorf("Expected id 123, got %d", builder.id)
	}
	if builder.state != 1 {
		t.Errorf("Expected state 1, got %d", builder.state)
	}
	if builder.x != 100 {
		t.Errorf("Expected x 100, got %d", builder.x)
	}
	if builder.y != 200 {
		t.Errorf("Expected y 200, got %d", builder.y)
	}
	if builder.delay != 1000 {
		t.Errorf("Expected delay 1000, got %d", builder.delay)
	}
	if builder.direction != 1 {
		t.Errorf("Expected direction 1, got %d", builder.direction)
	}
	if builder.eventState != 2 {
		t.Errorf("Expected eventState 2, got %d", builder.eventState)
	}
}

func TestModelBuilder_UpdateTime(t *testing.T) {
	f := field.NewBuilder(1, 2, 100000000).Build()
	builder := NewModelBuilder(f, 1001, "test-reactor")
	originalTime := builder.updateTime

	// Wait a tiny bit to ensure time changes
	time.Sleep(time.Millisecond)

	builder.UpdateTime()

	if !builder.updateTime.After(originalTime) {
		t.Error("UpdateTime() should update the updateTime to a later time")
	}
}

func TestModelBuilder_Classification(t *testing.T) {
	f := field.NewBuilder(1, 2, 100000000).Build()
	builder := NewModelBuilder(f, 1001, "test-reactor")

	if builder.Classification() != 1001 {
		t.Errorf("Expected Classification 1001, got %d", builder.Classification())
	}
}

// RestModel tests

func TestRestModel_GetID(t *testing.T) {
	tests := []struct {
		name     string
		id       uint32
		expected string
	}{
		{"zero", 0, "0"},
		{"typical id", 12345, "12345"},
		{"large id", 4294967295, "4294967295"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := RestModel{Id: tt.id}
			if rm.GetID() != tt.expected {
				t.Errorf("Expected GetID '%s', got '%s'", tt.expected, rm.GetID())
			}
		})
	}
}

func TestRestModel_GetName(t *testing.T) {
	rm := RestModel{}
	if rm.GetName() != "reactors" {
		t.Errorf("Expected GetName 'reactors', got '%s'", rm.GetName())
	}
}

func TestRestModel_SetID_Valid(t *testing.T) {
	tests := []struct {
		name     string
		idStr    string
		expected uint32
	}{
		{"zero", "0", 0},
		{"typical id", "12345", 12345},
		{"large id", "4294967295", 4294967295},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := RestModel{}
			err := rm.SetID(tt.idStr)
			if err != nil {
				t.Fatalf("SetID returned error: %v", err)
			}
			if rm.Id != tt.expected {
				t.Errorf("Expected Id %d, got %d", tt.expected, rm.Id)
			}
		})
	}
}

func TestRestModel_SetID_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		idStr string
	}{
		{"non-numeric", "invalid"},
		{"empty", ""},
		{"float", "123.45"},
		{"negative", "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := RestModel{}
			err := rm.SetID(tt.idStr)
			if err == nil {
				t.Error("SetID expected error for invalid input, got nil")
			}
		})
	}
}

func TestExtract(t *testing.T) {
	rm := RestModel{
		Id:             123,
		WorldId:        1,
		ChannelId:      2,
		MapId:          100000000,
		Classification: 1001,
		Name:           "test-reactor",
		State:          1,
		EventState:     2,
		X:              100,
		Y:              200,
		Delay:          1000,
		Direction:      1,
	}

	model, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if model.Id() != rm.Id {
		t.Errorf("Expected Id %d, got %d", rm.Id, model.Id())
	}
	if model.WorldId() != rm.WorldId {
		t.Errorf("Expected WorldId %d, got %d", rm.WorldId, model.WorldId())
	}
	if model.ChannelId() != rm.ChannelId {
		t.Errorf("Expected ChannelId %d, got %d", rm.ChannelId, model.ChannelId())
	}
	if model.MapId() != rm.MapId {
		t.Errorf("Expected MapId %d, got %d", rm.MapId, model.MapId())
	}
	if model.Classification() != rm.Classification {
		t.Errorf("Expected Classification %d, got %d", rm.Classification, model.Classification())
	}
	if model.Name() != rm.Name {
		t.Errorf("Expected Name '%s', got '%s'", rm.Name, model.Name())
	}
	if model.State() != rm.State {
		t.Errorf("Expected State %d, got %d", rm.State, model.State())
	}
	if model.EventState() != rm.EventState {
		t.Errorf("Expected EventState %d, got %d", rm.EventState, model.EventState())
	}
	if model.X() != rm.X {
		t.Errorf("Expected X %d, got %d", rm.X, model.X())
	}
	if model.Y() != rm.Y {
		t.Errorf("Expected Y %d, got %d", rm.Y, model.Y())
	}
	if model.Delay() != rm.Delay {
		t.Errorf("Expected Delay %d, got %d", rm.Delay, model.Delay())
	}
	if model.Direction() != rm.Direction {
		t.Errorf("Expected Direction %d, got %d", rm.Direction, model.Direction())
	}
}

func TestExtract_EmptyRestModel(t *testing.T) {
	rm := RestModel{}

	model, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	// Should have zero values
	if model.Id() != 0 {
		t.Errorf("Expected Id 0, got %d", model.Id())
	}
	if model.Name() != "" {
		t.Errorf("Expected empty Name, got '%s'", model.Name())
	}
}
