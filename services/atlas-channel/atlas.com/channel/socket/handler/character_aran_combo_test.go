package handler

import (
	"atlas-channel/character/combo"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus/hooks/test"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// comboTestTenant returns a fresh tenant per call so the process-wide combo
// Mirror singleton cannot leak state between subtests.
func comboTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tn
}

func comboTestField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
}

func TestIdleWindowFromOptions(t *testing.T) {
	tests := []struct {
		name string
		opts map[string]interface{}
		want time.Duration
	}{
		{"v95 5000", map[string]interface{}{"idleResetMs": float64(5000)}, 5000 * time.Millisecond},
		{"v83 3000", map[string]interface{}{"idleResetMs": float64(3000)}, 3000 * time.Millisecond},
		{"int form", map[string]interface{}{"idleResetMs": 3000}, 3000 * time.Millisecond},
		{"absent", map[string]interface{}{}, 3000 * time.Millisecond},
		{"nil options", nil, 3000 * time.Millisecond},
		{"non-numeric", map[string]interface{}{"idleResetMs": "soon"}, 3000 * time.Millisecond},
		{"zero", map[string]interface{}{"idleResetMs": float64(0)}, 3000 * time.Millisecond},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := idleWindowFromOptions(tc.opts); got != tc.want {
				t.Errorf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestAranComboAdvance(t *testing.T) {
	l, _ := test.NewNullLogger()
	tn := comboTestTenant(t)
	f := comboTestField()
	el := combo.NewEligibility(skill3.AranStage1ComboAbilityId, 5, 5)
	now := time.Unix(2000, 0)
	opts := map[string]interface{}{"idleResetMs": float64(3000)}

	newDeps := func(eligible bool, seedErr error, seeds *int, announced *[]uint32) aranComboDeps {
		return aranComboDeps{
			eligibility: func(uint32) (combo.Eligibility, bool) { return el, eligible },
			seed: func(combo.Eligibility, uint32) error {
				*seeds++
				return seedErr
			},
			announce: func(c uint32) error {
				*announced = append(*announced, c)
				return nil
			},
			now: func() time.Time { return now },
		}
	}

	t.Run("ineligible is a silent no-op", func(t *testing.T) {
		seeds, announced := 0, []uint32{}
		aranComboAdvance(l, newDeps(false, nil, &seeds, &announced), tn, 11, f, opts)
		if seeds != 0 || len(announced) != 0 {
			t.Fatalf("want no emissions, got seeds=%d announced=%v", seeds, announced)
		}
	})

	t.Run("first hit seeds and announces 1", func(t *testing.T) {
		seeds, announced := 0, []uint32{}
		d := newDeps(true, nil, &seeds, &announced)
		aranComboAdvance(l, d, tn, 12, f, opts)
		if seeds != 1 {
			t.Errorf("want exactly 1 seed, got %d", seeds)
		}
		if len(announced) != 1 || announced[0] != 1 {
			t.Errorf("want announce [1], got %v", announced)
		}
	})

	t.Run("second hit advances without re-seeding", func(t *testing.T) {
		seeds, announced := 0, []uint32{}
		d := newDeps(true, nil, &seeds, &announced)
		aranComboAdvance(l, d, tn, 13, f, opts)
		aranComboAdvance(l, d, tn, 13, f, opts)
		if seeds != 1 {
			t.Errorf("want exactly 1 seed across two hits, got %d", seeds)
		}
		if len(announced) != 2 || announced[1] != 2 {
			t.Errorf("want announce [1 2], got %v", announced)
		}
	})

	t.Run("seed failure still advances the count", func(t *testing.T) {
		seeds, announced := 0, []uint32{}
		d := newDeps(true, errors.New("broker down"), &seeds, &announced)
		aranComboAdvance(l, d, tn, 14, f, opts)
		if len(announced) != 1 || announced[0] != 1 {
			t.Errorf("combo bookkeeping never fails the action: want announce [1], got %v", announced)
		}
	})

	t.Run("at cap the count holds and no second seed fires", func(t *testing.T) {
		seeds, announced := 0, []uint32{}
		d := newDeps(true, nil, &seeds, &announced)
		aranComboAdvance(l, d, tn, 15, f, opts)
		for i := 0; i < 3; i++ {
			aranComboAdvance(l, d, tn, 15, f, opts)
		}
		if seeds != 1 {
			t.Errorf("want exactly 1 seed, got %d", seeds)
		}
		if len(announced) != 4 || announced[3] != 4 {
			t.Errorf("want announce [1 2 3 4], got %v", announced)
		}
	})
}
