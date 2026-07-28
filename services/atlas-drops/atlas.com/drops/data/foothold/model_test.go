package foothold

import "testing"

// TestModelLandingY pins the floor-height computation used to snap a drop to
// the foothold below it. Values mirror atlas-data's calcYOnFoothold so the
// drop rests exactly where the client draws the floor.
func TestModelLandingY(t *testing.T) {
	// tol allows for the integer truncation inherent in the trig-based slope
	// computation ported verbatim from atlas-data's calcYOnFoothold: a 45°
	// midpoint resolves to 149/151 rather than an idealized 150. Flat and
	// endpoint cases are exact (tol 0).
	tests := []struct {
		name   string
		fh     Model
		x      int16
		wantY  int16
		tol    int16
		wantOk bool
	}{
		{name: "flat foothold", fh: Model{x1: 0, y1: 100, x2: 200, y2: 100}, x: 137, wantY: 100, tol: 0, wantOk: true},
		{name: "downward slope midpoint", fh: Model{x1: 0, y1: 100, x2: 100, y2: 200}, x: 50, wantY: 150, tol: 1, wantOk: true},
		{name: "upward slope midpoint", fh: Model{x1: 0, y1: 200, x2: 100, y2: 100}, x: 50, wantY: 150, tol: 1, wantOk: true},
		{name: "at left endpoint", fh: Model{x1: 10, y1: 300, x2: 110, y2: 300}, x: 10, wantY: 300, tol: 0, wantOk: true},
		{name: "at right endpoint", fh: Model{x1: 10, y1: 300, x2: 110, y2: 300}, x: 110, wantY: 300, tol: 0, wantOk: true},
		{name: "x left of segment", fh: Model{x1: 0, y1: 100, x2: 100, y2: 100}, x: -5, wantOk: false},
		{name: "x right of segment", fh: Model{x1: 0, y1: 100, x2: 100, y2: 100}, x: 250, wantOk: false},
		{name: "wall (vertical) has no floor", fh: Model{x1: 50, y1: 0, x2: 50, y2: 400}, x: 50, wantOk: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotY, gotOk := tc.fh.LandingY(tc.x)
			if gotOk != tc.wantOk {
				t.Fatalf("ok = %v, want %v", gotOk, tc.wantOk)
			}
			if tc.wantOk {
				diff := gotY - tc.wantY
				if diff < 0 {
					diff = -diff
				}
				if diff > tc.tol {
					t.Errorf("landing y = %d, want %d (±%d)", gotY, tc.wantY, tc.tol)
				}
			}
		})
	}
}
