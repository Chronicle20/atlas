package reactor

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TestTransformRoundTrip asserts Extract(Transform(m)) == m over every field
// RestModel carries. UpdateTime is excluded: RestModel has no field for it,
// so it cannot round-trip (see rest.go:Extract, model.go Model.updateTime;
// recorded in docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md).
func TestTransformRoundTrip(t *testing.T) {
	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(3)).SetInstance(uuid.New()).Build()

	m, err := NewBuilder(f, 4, "reactorName").
		SetId(5).
		SetState(6).
		SetEventState(7).
		SetPosition(8, 9).
		SetDelay(10).
		SetDirection(11).
		Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if got.Id() != m.Id() {
		t.Errorf("Id mismatch. Expected %v, got %v", m.Id(), got.Id())
	}
	if got.WorldId() != m.WorldId() {
		t.Errorf("WorldId mismatch. Expected %v, got %v", m.WorldId(), got.WorldId())
	}
	if got.ChannelId() != m.ChannelId() {
		t.Errorf("ChannelId mismatch. Expected %v, got %v", m.ChannelId(), got.ChannelId())
	}
	if got.MapId() != m.MapId() {
		t.Errorf("MapId mismatch. Expected %v, got %v", m.MapId(), got.MapId())
	}
	if got.Field().Instance() != m.Field().Instance() {
		t.Errorf("Instance mismatch. Expected %v, got %v", m.Field().Instance(), got.Field().Instance())
	}
	if got.Classification() != m.Classification() {
		t.Errorf("Classification mismatch. Expected %v, got %v", m.Classification(), got.Classification())
	}
	if got.Name() != m.Name() {
		t.Errorf("Name mismatch. Expected %v, got %v", m.Name(), got.Name())
	}
	if got.State() != m.State() {
		t.Errorf("State mismatch. Expected %v, got %v", m.State(), got.State())
	}
	if got.EventState() != m.EventState() {
		t.Errorf("EventState mismatch. Expected %v, got %v", m.EventState(), got.EventState())
	}
	if got.Delay() != m.Delay() {
		t.Errorf("Delay mismatch. Expected %v, got %v", m.Delay(), got.Delay())
	}
	if got.Direction() != m.Direction() {
		t.Errorf("Direction mismatch. Expected %v, got %v", m.Direction(), got.Direction())
	}
	if got.X() != m.X() {
		t.Errorf("X mismatch. Expected %v, got %v", m.X(), got.X())
	}
	if got.Y() != m.Y() {
		t.Errorf("Y mismatch. Expected %v, got %v", m.Y(), got.Y())
	}
}
