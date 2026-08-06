package shops

import (
	"atlas-npc/asset"
	"testing"
)

const perfectPitch = uint32(4310000)

// etcAsset builds an asset.Model through the exported production path.
// asset.Model has no Builder (the package is model.go + rest.go only), and
// CLAUDE.md forbids *_testhelpers.go with test-only constructors.
// Model.Quantity() returns the stored value only when HasQuantity() is true,
// which holds for ETC ids like 4310000 via IsStackable() (asset/model.go:127-140).
func etcAsset(t *testing.T, slot int16, templateId uint32, quantity uint32) asset.Model {
	t.Helper()
	a, err := asset.Extract(asset.BaseRestModel{
		Slot:       slot,
		TemplateId: templateId,
		Quantity:   quantity,
	})
	if err != nil {
		t.Fatalf("failed to build asset: %v", err)
	}
	if a.Quantity() != quantity {
		t.Fatalf("asset quantity did not survive Extract: got %d want %d", a.Quantity(), quantity)
	}
	return a
}

func TestPlanTokenSpend(t *testing.T) {
	tests := []struct {
		name          string
		assets        func(t *testing.T) []asset.Model
		cost          uint32
		wantDraws     []tokenDraw
		wantAvailable uint64
	}{
		{
			name: "exact single slot",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{etcAsset(t, 3, perfectPitch, 60)}
			},
			cost:          60,
			wantDraws:     []tokenDraw{{slot: 3, quantity: 60}},
			wantAvailable: 60,
		},
		{
			name: "cost spans two slots and the second is drawn partially",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{
					etcAsset(t, 3, perfectPitch, 60),
					etcAsset(t, 7, perfectPitch, 55),
				}
			},
			cost:          100,
			wantDraws:     []tokenDraw{{slot: 3, quantity: 60}, {slot: 7, quantity: 40}},
			wantAvailable: 115,
		},
		{
			name: "cost spans three slots",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{
					etcAsset(t, 1, perfectPitch, 40),
					etcAsset(t, 2, perfectPitch, 40),
					etcAsset(t, 3, perfectPitch, 40),
				}
			},
			cost: 100,
			wantDraws: []tokenDraw{
				{slot: 1, quantity: 40},
				{slot: 2, quantity: 40},
				{slot: 3, quantity: 20},
			},
			wantAvailable: 120,
		},
		{
			name: "cost exceeds total held returns a short plan and the true total",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{
					etcAsset(t, 2, perfectPitch, 10),
					etcAsset(t, 5, perfectPitch, 15),
				}
			},
			cost:          100,
			wantDraws:     []tokenDraw{{slot: 2, quantity: 10}, {slot: 5, quantity: 15}},
			wantAvailable: 25,
		},
		{
			name: "zero-quantity slots are skipped",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{
					etcAsset(t, 1, perfectPitch, 0),
					etcAsset(t, 4, perfectPitch, 30),
				}
			},
			cost:          20,
			wantDraws:     []tokenDraw{{slot: 4, quantity: 20}},
			wantAvailable: 30,
		},
		{
			name: "non-matching template ids are ignored",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{
					etcAsset(t, 1, 4000000, 999),
					etcAsset(t, 2, perfectPitch, 12),
				}
			},
			cost:          12,
			wantDraws:     []tokenDraw{{slot: 2, quantity: 12}},
			wantAvailable: 12,
		},
		{
			name: "draws are ascending by slot regardless of input order",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{
					etcAsset(t, 9, perfectPitch, 50),
					etcAsset(t, 2, perfectPitch, 50),
					etcAsset(t, 5, perfectPitch, 50),
				}
			},
			cost: 110,
			wantDraws: []tokenDraw{
				{slot: 2, quantity: 50},
				{slot: 5, quantity: 50},
				{slot: 9, quantity: 10},
			},
			wantAvailable: 150,
		},
		{
			name: "zero cost draws nothing but still reports what is held",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{etcAsset(t, 1, perfectPitch, 7)}
			},
			cost:          0,
			wantDraws:     []tokenDraw{},
			wantAvailable: 7,
		},
		{
			name: "empty compartment",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{}
			},
			cost:          5,
			wantDraws:     []tokenDraw{},
			wantAvailable: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			draws, available := planTokenSpend(tt.assets(t), perfectPitch, tt.cost)

			if available != tt.wantAvailable {
				t.Errorf("available: got %d want %d", available, tt.wantAvailable)
			}
			if len(draws) != len(tt.wantDraws) {
				t.Fatalf("draws: got %d entries %v, want %d entries %v",
					len(draws), draws, len(tt.wantDraws), tt.wantDraws)
			}
			for i := range draws {
				if draws[i] != tt.wantDraws[i] {
					t.Errorf("draws[%d]: got %+v want %+v", i, draws[i], tt.wantDraws[i])
				}
			}
		})
	}
}
