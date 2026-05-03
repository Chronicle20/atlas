package _map

import (
	"atlas-data/map/monster"
	monstertpl "atlas-data/monster"
	"atlas-data/point"
	"errors"
	"testing"
)

var errTestMissing = errors.New("test: template missing")

func snapTestTree() FootholdTreeRestModel {
	tree := NewFootholdTree(-2000, -2000, 2000, 2000)
	footholds := []FootholdRestModel{
		{Id: 10, First: &point.RestModel{X: -200, Y: 100}, Second: &point.RestModel{X: 200, Y: 100}}, // flat
		{Id: 11, First: &point.RestModel{X: 200, Y: 100}, Second: &point.RestModel{X: 400, Y: 200}},  // down-slope
	}
	return *tree.Insert(footholds)
}

func groundLookup(template uint32) (monstertpl.RestModel, error) {
	return monstertpl.RestModel{Id: template, Flying: false, Swimming: false}, nil
}
func flyingLookup(template uint32) (monstertpl.RestModel, error) {
	return monstertpl.RestModel{Id: template, Flying: true}, nil
}
func swimmingLookup(template uint32) (monstertpl.RestModel, error) {
	return monstertpl.RestModel{Id: template, Swimming: true}, nil
}
func errLookup(template uint32) (monstertpl.RestModel, error) {
	return monstertpl.RestModel{}, errTestMissing
}

func TestSnapToGround(t *testing.T) {
	tree := snapTestTree()

	tests := []struct {
		name        string
		sp          monster.RestModel
		lookup      templateLookup
		wantYExact  *int16
		wantYMin    int16
		wantYMax    int16
		useTolerant bool
	}{
		{
			// FH-set branch snaps to the foothold surface y exactly. The
			// 1-px-above-surface adjustment that prevents v83 client
			// fall-through happens in atlas-channel's wire-packet snap
			// (data/map.SnapMobPosition); atlas-data returns the raw surface
			// here so there's a single source of truth for the snap invariant.
			name:       "fh_set_flat_corrects_y",
			sp:         monster.RestModel{Template: 100100, X: 0, Y: 80, FH: 10},
			lookup:     groundLookup,
			wantYExact: ptrInt16(100),
		},
		{
			name:        "fh_set_slope_corrects_y",
			sp:          monster.RestModel{Template: 100100, X: 300, Y: 80, FH: 11},
			lookup:      groundLookup,
			wantYMin:    145,
			wantYMax:    155,
			useTolerant: true,
		},
		{
			name:       "fh_set_missing_leaves_y",
			sp:         monster.RestModel{Template: 100100, X: 0, Y: 80, FH: 9999},
			lookup:     groundLookup,
			wantYExact: ptrInt16(80),
		},
		{
			name:       "fh_zero_flying_mob_leaves_y",
			sp:         monster.RestModel{Template: 2230000, X: 0, Y: -300, FH: 0},
			lookup:     flyingLookup,
			wantYExact: ptrInt16(-300),
		},
		{
			name:       "fh_zero_swimming_mob_leaves_y",
			sp:         monster.RestModel{Template: 2230100, X: 0, Y: 50, FH: 0},
			lookup:     swimmingLookup,
			wantYExact: ptrInt16(50),
		},
		{
			name:       "fh_zero_ground_mob_finds_below",
			sp:         monster.RestModel{Template: 100100, X: 0, Y: 80, FH: 0},
			lookup:     groundLookup,
			wantYExact: ptrInt16(99),
		},
		{
			name:       "fh_zero_no_foothold_below_leaves_y",
			sp:         monster.RestModel{Template: 100100, X: 9999, Y: 80, FH: 0},
			lookup:     groundLookup,
			wantYExact: ptrInt16(80),
		},
		{
			name:       "fh_zero_template_lookup_err_leaves_y",
			sp:         monster.RestModel{Template: 100100, X: 0, Y: 80, FH: 0},
			lookup:     errLookup,
			wantYExact: ptrInt16(80),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := snapToGround(tree, tt.sp, tt.lookup)
			if tt.useTolerant {
				if out.Y < tt.wantYMin || out.Y > tt.wantYMax {
					t.Fatalf("Y=%d, want in [%d, %d]", out.Y, tt.wantYMin, tt.wantYMax)
				}
				return
			}
			if out.Y != *tt.wantYExact {
				t.Fatalf("Y=%d, want %d", out.Y, *tt.wantYExact)
			}
		})
	}
}

func TestSnapToGroundIdempotent(t *testing.T) {
	tree := snapTestTree()
	sp := monster.RestModel{Template: 100100, X: 0, Y: 80, FH: 10}
	once := snapToGround(tree, sp, groundLookup)
	twice := snapToGround(tree, once, groundLookup)
	if once.Y != twice.Y {
		t.Fatalf("idempotency broken: once=%d, twice=%d", once.Y, twice.Y)
	}
}

func ptrInt16(v int16) *int16 { return &v }
