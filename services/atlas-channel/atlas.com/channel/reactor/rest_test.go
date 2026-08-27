package reactor

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract for every field RestModel can carry.
//
// updateTime is asserted separately, not via reflect.DeepEqual over the
// whole Model: RestModel has no UpdateTime field (reactor/rest.go:14-28), so
// Extract can never restore it (reactor/rest.go:48-61), yet the builder sets
// it to time.Now() on construction (reactor/builder.go:26-33). A Model built
// through NewBuilder therefore always carries a non-zero updateTime that
// Transform -> Extract cannot round-trip; the field is dropped to its zero
// value. Recorded in handwork-notes.md under "Batch channel-d".
func TestTransformRoundTrip(t *testing.T) {
	f := field.NewBuilder(1, 2, 100).SetInstance(uuid.New()).Build()

	m, err := NewBuilder(f, 300, "reactor-name").
		SetId(400).
		SetState(5).
		SetEventState(6).
		SetPosition(7, 8).
		SetDelay(9).
		SetDirection(10).
		Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	m2, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if m2.Id() != m.Id() {
		t.Errorf("Id mismatch. Expected %v, got %v", m.Id(), m2.Id())
	}
	if m2.WorldId() != m.WorldId() {
		t.Errorf("WorldId mismatch. Expected %v, got %v", m.WorldId(), m2.WorldId())
	}
	if m2.ChannelId() != m.ChannelId() {
		t.Errorf("ChannelId mismatch. Expected %v, got %v", m.ChannelId(), m2.ChannelId())
	}
	if m2.MapId() != m.MapId() {
		t.Errorf("MapId mismatch. Expected %v, got %v", m.MapId(), m2.MapId())
	}
	if m2.Instance() != m.Instance() {
		t.Errorf("Instance mismatch. Expected %v, got %v", m.Instance(), m2.Instance())
	}
	if m2.Classification() != m.Classification() {
		t.Errorf("Classification mismatch. Expected %v, got %v", m.Classification(), m2.Classification())
	}
	if m2.Name() != m.Name() {
		t.Errorf("Name mismatch. Expected %v, got %v", m.Name(), m2.Name())
	}
	if m2.State() != m.State() {
		t.Errorf("State mismatch. Expected %v, got %v", m.State(), m2.State())
	}
	if m2.EventState() != m.EventState() {
		t.Errorf("EventState mismatch. Expected %v, got %v", m.EventState(), m2.EventState())
	}
	if m2.Delay() != m.Delay() {
		t.Errorf("Delay mismatch. Expected %v, got %v", m.Delay(), m2.Delay())
	}
	if m2.Direction() != m.Direction() {
		t.Errorf("Direction mismatch. Expected %v, got %v", m.Direction(), m2.Direction())
	}
	if m2.X() != m.X() {
		t.Errorf("X mismatch. Expected %v, got %v", m.X(), m2.X())
	}
	if m2.Y() != m.Y() {
		t.Errorf("Y mismatch. Expected %v, got %v", m.Y(), m2.Y())
	}

	// updateTime is NOT asserted equal — it is genuinely lossy (see doc
	// comment above). It must, however, be dropped to the zero value, not
	// silently invented.
	if !m2.UpdateTime().IsZero() {
		t.Errorf("expected UpdateTime to be zero after round trip (RestModel carries no such field), got %v", m2.UpdateTime())
	}
}
