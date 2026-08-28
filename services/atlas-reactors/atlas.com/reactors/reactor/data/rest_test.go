package data

import (
	"atlas-reactors/reactor/data/area"
	"atlas-reactors/reactor/data/item"
	"atlas-reactors/reactor/data/point"
	"atlas-reactors/reactor/data/state"
	"reflect"
	"testing"
)

func TestExtractTouchFields(t *testing.T) {
	tests := []struct {
		name  string
		input RestModel
	}{
		{
			name: "flag set with areas",
			input: RestModel{
				ActivateByTouch: true,
				TouchAreaInfo: map[int8]area.RestModel{
					0: {TL: point.RestModel{X: -53, Y: 24}, BR: point.RestModel{X: 62, Y: 69}},
				},
			},
		},
		{
			name:  "fields absent",
			input: RestModel{Name: "x"},
		},
		{
			name: "unknown state",
			input: RestModel{
				ActivateByTouch: true,
				TouchAreaInfo: map[int8]area.RestModel{
					0: {TL: point.RestModel{X: -53, Y: 24}, BR: point.RestModel{X: 62, Y: 69}},
				},
			},
		},
	}

	// case 1: flag set with areas
	m, err := Extract(tests[0].input)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if m.ActivateByTouch() != true {
		t.Fatalf("ActivateByTouch() = false, want true")
	}
	a, ok := m.TouchArea(0)
	if !ok {
		t.Fatalf("TouchArea(0) ok = false, want true")
	}
	if a.TL().X() != -53 {
		t.Fatalf("TouchArea(0).TL().X() = %d, want -53", a.TL().X())
	}
	if a.TL().Y() != 24 {
		t.Fatalf("TouchArea(0).TL().Y() = %d, want 24", a.TL().Y())
	}
	if a.BR().X() != 62 {
		t.Fatalf("TouchArea(0).BR().X() = %d, want 62", a.BR().X())
	}
	if a.BR().Y() != 69 {
		t.Fatalf("TouchArea(0).BR().Y() = %d, want 69", a.BR().Y())
	}

	// case 2: fields absent (FR-12)
	m2, err := Extract(tests[1].input)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if m2.ActivateByTouch() != false {
		t.Fatalf("ActivateByTouch() = true, want false")
	}
	if _, ok := m2.TouchArea(0); ok {
		t.Fatalf("TouchArea(0) ok = true, want false on nil map")
	}

	// case 3: unknown state
	m3, err := Extract(tests[2].input)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if _, ok := m3.TouchArea(1); ok {
		t.Fatalf("TouchArea(1) ok = true, want false for unknown state")
	}
}

func TestModelJSONRoundTripTouchFields(t *testing.T) {
	rm := RestModel{
		ActivateByTouch: true,
		TouchAreaInfo: map[int8]area.RestModel{
			0: {TL: point.RestModel{X: -53, Y: 24}, BR: point.RestModel{X: 62, Y: 69}},
		},
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	b, err := m.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	var m2 Model
	if err := m2.UnmarshalJSON(b); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	if m2.ActivateByTouch() != true {
		t.Fatalf("round-tripped ActivateByTouch() = false, want true")
	}
	a, ok := m2.TouchArea(0)
	if !ok {
		t.Fatalf("round-tripped TouchArea(0) ok = false, want true")
	}
	if a.TL().X() != -53 {
		t.Fatalf("round-tripped TouchArea(0).TL().X() = %d, want -53", a.TL().X())
	}
	if a.TL().Y() != 24 {
		t.Fatalf("round-tripped TouchArea(0).TL().Y() = %d, want 24", a.TL().Y())
	}
	if a.BR().X() != 62 {
		t.Fatalf("round-tripped TouchArea(0).BR().X() = %d, want 62", a.BR().X())
	}
	if a.BR().Y() != 69 {
		t.Fatalf("round-tripped TouchArea(0).BR().Y() = %d, want 69", a.BR().Y())
	}
}

// TestTransformRoundTrip verifies Transform is the faithful inverse of
// Extract: Extract(Transform(m)) reproduces m.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Name:            "Reactor",
		TL:              point.RestModel{X: -53, Y: 24},
		BR:              point.RestModel{X: 62, Y: 69},
		ActivateByTouch: true,
		TouchAreaInfo: map[int8]area.RestModel{
			0: {TL: point.RestModel{X: -53, Y: 24}, BR: point.RestModel{X: 62, Y: 69}},
		},
		StateInfo: map[int8][]state.RestModel{
			0: {
				{
					Type:         1,
					ReactorItem:  &item.RestModel{ItemId: 2000000, Quantity: 5},
					ActiveSkills: []uint32{1000},
					NextState:    1,
				},
			},
		},
		TimeoutInfo:          map[int8]int32{0: 5000},
		TimeoutNextStateInfo: map[int8]int8{0: 1},
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rm2, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (round trip) failed: %v", err)
	}

	if !reflect.DeepEqual(m2, m) {
		t.Errorf("round trip mismatch. want %+v, got %+v", m, m2)
	}
}
