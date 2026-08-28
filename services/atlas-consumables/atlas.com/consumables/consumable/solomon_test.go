package consumable

import (
	"atlas-consumables/character"
	consumable3 "atlas-consumables/data/consumable"
	"context"
	"errors"
	"testing"

	charmock "atlas-consumables/character/mock"
	compmock "atlas-consumables/compartment/mock"
	consumablemock "atlas-consumables/data/consumable/mock"
	mapcharmock "atlas-consumables/map/character/mock"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	fieldc "github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map2 "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// extractSolomonItem builds a data/consumable Model through the package's own
// exported Extract, so no test-only constructor is introduced (CLAUDE.md
// test-helper rule).
func extractSolomonItem(t *testing.T, spec map[consumable3.SpecType]int32, maxLevel uint32) consumable3.Model {
	t.Helper()
	m, err := consumable3.Extract(consumable3.RestModel{Id: 2370000, MaxLevel: maxLevel, Spec: spec})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	return m
}

// extractSolomonCharacter builds a character Model through the package's own
// exported Extract.
func extractSolomonCharacter(t *testing.T, level byte, gachaponExperience uint32) character.Model {
	t.Helper()
	m, err := character.Extract(character.RestModel{Level: level, GachaponExperience: gachaponExperience})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	return m
}

// solomonConsumeItemCall records one compartment.ConsumeItem invocation.
type solomonConsumeItemCall struct {
	characterId   uint32
	inventoryType inventory2.Type
	slot          int16
}

// solomonCreditCall records one CreditStoredExperience invocation.
type solomonCreditCall struct {
	characterId uint32
	amount      uint32
	reason      string
}

// solomonHarness wires the package mocks into a solomonDeps and captures
// every outbound call. onError records the ConsumeError route without
// needing a live Kafka broker.
type solomonHarness struct {
	deps         solomonDeps
	consumeItems []solomonConsumeItemCall
	credits      []solomonCreditCall
	errors       []error
}

func newSolomonHarness(t *testing.T, ci consumable3.Model, dataErr error, c character.Model) *solomonHarness {
	t.Helper()
	f := fieldc.NewBuilder(world.Id(0), channel.Id(0), _map2.Id(100000000)).Build()
	h := &solomonHarness{}
	h.deps = solomonDeps{
		data: &consumablemock.ProcessorMock{
			GetByIdFunc: func(uint32) (consumable3.Model, error) { return ci, dataErr },
		},
		fields: &mapcharmock.ProcessorMock{
			GetMapFunc: func(uint32) (fieldc.Model, error) { return f, nil },
		},
		compartment: &compmock.ProcessorMock{
			ConsumeItemFunc: func(characterId uint32, it inventory2.Type, _ uuid.UUID, slot int16) error {
				h.consumeItems = append(h.consumeItems, solomonConsumeItemCall{characterId, it, slot})
				return nil
			},
		},
		character: &charmock.ProcessorMock{
			GetByIdFunc: func(_ ...model.Decorator[character.Model]) func(uint32) (character.Model, error) {
				return func(uint32) (character.Model, error) { return c, nil }
			},
			CreditStoredExperienceFunc: func(_ fieldc.Model, characterId uint32, amount uint32, reason string) error {
				h.credits = append(h.credits, solomonCreditCall{characterId, amount, reason})
				return nil
			},
		},
		onError: func(err error) error {
			h.errors = append(h.errors, err)
			return err
		},
	}
	return h
}

// TestConsumeSolomon pins FR-6: every rejection releases the Writ instead of
// destroying it, and only an eligible Writ commits and credits.
func TestConsumeSolomon(t *testing.T) {
	tests := []struct {
		name          string
		exp           int32
		expAbsent     bool
		maxLevel      uint32
		level         byte
		balance       uint32
		wantConsume   bool
		wantCredit    bool
		wantCreditAmt int32
		wantErr       bool
	}{
		{
			name:          "eligible",
			exp:           3000,
			maxLevel:      200,
			level:         30,
			balance:       0,
			wantConsume:   true,
			wantCredit:    true,
			wantCreditAmt: 3000,
		},
		{
			name:          "maxLevel absent means no upper bound",
			exp:           3000,
			maxLevel:      0,
			level:         200,
			balance:       0,
			wantConsume:   true,
			wantCredit:    true,
			wantCreditAmt: 3000,
		},
		{
			name:     "level above maxLevel",
			exp:      3000,
			maxLevel: 20,
			level:    30,
			balance:  0,
			wantErr:  true,
		},
		{
			name:          "level exactly at maxLevel",
			exp:           3000,
			maxLevel:      30,
			level:         30,
			balance:       0,
			wantConsume:   true,
			wantCredit:    true,
			wantCreditAmt: 3000,
		},
		{
			name:     "balance already non-zero",
			exp:      3000,
			maxLevel: 200,
			level:    30,
			balance:  1200,
			wantErr:  true,
		},
		{
			name:      "spec/exp absent",
			expAbsent: true,
			maxLevel:  200,
			level:     30,
			balance:   0,
			wantErr:   true,
		},
		{
			name:     "spec/exp negative",
			exp:      -5,
			maxLevel: 200,
			level:    30,
			balance:  0,
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec := map[consumable3.SpecType]int32{}
			if !tc.expAbsent {
				spec[consumable3.SpecTypeExperience] = tc.exp
			}
			ci := extractSolomonItem(t, spec, tc.maxLevel)
			c := extractSolomonCharacter(t, tc.level, tc.balance)
			h := newSolomonHarness(t, ci, nil, c)

			err := consumeSolomon(logrus.New(), context.Background(), h.deps, uuid.New(), 555, 3, 2370000)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("consumeSolomon: err = nil, want rejection error")
				}
			} else if err != nil {
				t.Fatalf("consumeSolomon: %v", err)
			}

			if tc.wantConsume {
				if len(h.consumeItems) != 1 {
					t.Fatalf("ConsumeItem call count = %d, want 1", len(h.consumeItems))
				}
				if got := h.consumeItems[0].inventoryType; got != inventory2.TypeValueUse {
					t.Errorf("ConsumeItem inventoryType = %d, want TypeValueUse (%d)", got, inventory2.TypeValueUse)
				}
				if h.consumeItems[0].slot != 3 {
					t.Errorf("ConsumeItem slot = %d, want 3", h.consumeItems[0].slot)
				}
			} else {
				if len(h.consumeItems) != 0 {
					t.Errorf("ConsumeItem call count = %d, want 0 — the Writ must stay in the inventory", len(h.consumeItems))
				}
			}

			if tc.wantCredit {
				if len(h.credits) != 1 {
					t.Fatalf("CreditStoredExperience call count = %d, want 1", len(h.credits))
				}
				if h.credits[0].amount != uint32(tc.wantCreditAmt) {
					t.Errorf("CreditStoredExperience amount = %d, want %d", h.credits[0].amount, tc.wantCreditAmt)
				}
				if h.credits[0].reason != "SOLOMON_ITEM" {
					t.Errorf("CreditStoredExperience reason = %q, want SOLOMON_ITEM", h.credits[0].reason)
				}
			} else {
				if len(h.credits) != 0 {
					t.Errorf("CreditStoredExperience call count = %d, want 0", len(h.credits))
				}
			}

			if tc.wantErr {
				if len(h.errors) != 1 {
					t.Errorf("onError call count = %d, want 1", len(h.errors))
				}
			} else {
				if len(h.errors) != 0 {
					t.Errorf("onError = %v, want none", h.errors)
				}
			}
		})
	}
}

// TestConsumeSolomonDataReadFailure pins FR-6: a data-read failure returns
// the Writ, it is never consumed for a failed read.
func TestConsumeSolomonDataReadFailure(t *testing.T) {
	wantErr := errors.New("consumables 404")
	h := newSolomonHarness(t, consumable3.Model{}, wantErr, extractSolomonCharacter(t, 30, 0))

	err := consumeSolomon(logrus.New(), context.Background(), h.deps, uuid.New(), 555, 3, 2370000)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if len(h.errors) != 1 {
		t.Fatalf("onError call count = %d, want 1", len(h.errors))
	}
	if len(h.consumeItems) != 0 {
		t.Errorf("ConsumeItem call count = %d, want 0", len(h.consumeItems))
	}
}

// TestRoutesToSolomon pins the classification gate for the Writ of Solomon
// (is_exp_up_item, classification 237).
func TestRoutesToSolomon(t *testing.T) {
	tests := []struct {
		itemId item2.Id
		want   bool
	}{
		{2370000, true},
		{2370012, true},
		{2379999, true},
		{2369999, false},
		{2380000, false},
		{2000000, false},
	}
	for _, tc := range tests {
		if got := routesToSolomon(tc.itemId); got != tc.want {
			t.Errorf("routesToSolomon(%d) = %t, want %t", tc.itemId, got, tc.want)
		}
	}
}
