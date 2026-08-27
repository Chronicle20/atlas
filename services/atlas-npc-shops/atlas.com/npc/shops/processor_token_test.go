package shops

import (
	"atlas-npc/asset"
	"atlas-npc/character"
	"atlas-npc/commodities"
	"atlas-npc/compartment"
	inventory2 "atlas-npc/inventory"
	"atlas-npc/kafka/message"
	compartmentMessage "atlas-npc/kafka/message/compartment"
	"atlas-npc/kafka/message/shops"
	"encoding/json"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
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

const (
	testCharacterId = uint32(1234)
	// 2022503 is a USE item, so the destination compartment is TypeValueUse.
	testItemId = uint32(2022503)
)

func testProcessor(t *testing.T) *ProcessorImpl {
	t.Helper()
	l := logrus.New()
	l.SetOutput(io.Discard)
	return &ProcessorImpl{
		l:     l,
		compP: compartment.NewProcessor(),
	}
}

// testCharacter builds a character holding etcAssets in its ETC compartment
// and useAssets in a USE compartment of the given capacity.
func testCharacter(t *testing.T, etcAssets []asset.Model, useCapacity uint32, useAssets []asset.Model) character.Model {
	t.Helper()
	etcComp := compartment.NewBuilder(uuid.New(), testCharacterId, inventory.TypeValueETC, 24).
		SetAssets(etcAssets).
		Build()
	useComp := compartment.NewBuilder(uuid.New(), testCharacterId, inventory.TypeValueUse, useCapacity).
		SetAssets(useAssets).
		Build()
	inv := inventory2.NewBuilder(testCharacterId).
		SetCompartment(etcComp).
		SetCompartment(useComp).
		Build()
	return character.NewBuilder().
		SetId(testCharacterId).
		SetInventory(inv).
		Build()
}

func testCommodity(t *testing.T, tokenTemplateId uint32, tokenPrice uint32) commodities.Model {
	t.Helper()
	cm, err := commodities.NewBuilder().
		SetId(uuid.New()).
		SetNpcId(9000069).
		SetTemplateId(testItemId).
		SetTokenTemplateId(tokenTemplateId).
		SetTokenPrice(tokenPrice).
		Build()
	if err != nil {
		t.Fatalf("failed to build commodity: %v", err)
	}
	return cm
}

func decodeDestroy(t *testing.T, raw []byte) compartmentMessage.Command[compartmentMessage.DestroyCommandBody] {
	t.Helper()
	var c compartmentMessage.Command[compartmentMessage.DestroyCommandBody]
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("failed to decode destroy command: %v", err)
	}
	return c
}

func decodeCreate(t *testing.T, raw []byte) compartmentMessage.Command[compartmentMessage.CreateAssetCommandBody] {
	t.Helper()
	var c compartmentMessage.Command[compartmentMessage.CreateAssetCommandBody]
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("failed to decode create command: %v", err)
	}
	return c
}

func decodeStatusError(t *testing.T, raw []byte) shops.StatusEvent[shops.StatusEventErrorBody] {
	t.Helper()
	var e shops.StatusEvent[shops.StatusEventErrorBody]
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatalf("failed to decode status event: %v", err)
	}
	return e
}

func TestBuyWithTokensSufficientBalanceSpansSlots(t *testing.T) {
	p := testProcessor(t)
	buf := message.NewBuffer()
	c := testCharacter(t,
		[]asset.Model{
			etcAsset(t, 7, perfectPitch, 55),
			etcAsset(t, 3, perfectPitch, 60),
		},
		24, nil)
	cm := testCommodity(t, perfectPitch, 100)

	if err := p.buyWithTokens(buf)(c, cm, testItemId, 1); err != nil {
		t.Fatalf("buyWithTokens returned an error: %v", err)
	}

	all := buf.GetAll()
	assertSingleOk(t, buf)

	cmds := all[compartmentMessage.EnvCommandTopic]
	if len(cmds) != 3 {
		t.Fatalf("expected 2 destroys + 1 create, got %d commands", len(cmds))
	}

	d0 := decodeDestroy(t, cmds[0].Value)
	if d0.Type != compartmentMessage.CommandDestroy ||
		d0.CharacterId != testCharacterId ||
		d0.InventoryType != byte(inventory.TypeValueETC) ||
		d0.Body.Slot != 3 || d0.Body.Quantity != 60 {
		t.Errorf("destroy[0]: got %+v", d0)
	}

	d1 := decodeDestroy(t, cmds[1].Value)
	if d1.Body.Slot != 7 || d1.Body.Quantity != 40 {
		t.Errorf("destroy[1]: got slot %d quantity %d, want slot 7 quantity 40",
			d1.Body.Slot, d1.Body.Quantity)
	}

	cr := decodeCreate(t, cmds[2].Value)
	if cr.Type != compartmentMessage.CommandCreateAsset ||
		cr.InventoryType != byte(inventory.TypeValueUse) ||
		cr.Body.TemplateId != testItemId || cr.Body.Quantity != 1 {
		t.Errorf("create: got %+v", cr)
	}
}

func TestBuyWithTokensQuantityMultipliesCost(t *testing.T) {
	p := testProcessor(t)
	buf := message.NewBuffer()
	c := testCharacter(t, []asset.Model{etcAsset(t, 1, perfectPitch, 100)}, 24, nil)
	cm := testCommodity(t, perfectPitch, 5)

	if err := p.buyWithTokens(buf)(c, cm, testItemId, 3); err != nil {
		t.Fatalf("buyWithTokens returned an error: %v", err)
	}

	assertSingleOk(t, buf)

	cmds := buf.GetAll()[compartmentMessage.EnvCommandTopic]
	if len(cmds) != 2 {
		t.Fatalf("expected 1 destroy + 1 create, got %d commands", len(cmds))
	}
	if d := decodeDestroy(t, cmds[0].Value); d.Body.Quantity != 15 {
		t.Errorf("destroy quantity: got %d want 15 (tokenPrice 5 x quantity 3)", d.Body.Quantity)
	}
	if cr := decodeCreate(t, cmds[1].Value); cr.Body.Quantity != 3 {
		t.Errorf("create quantity: got %d want 3", cr.Body.Quantity)
	}
}

func TestBuyWithTokensRefusals(t *testing.T) {
	noAssets := func(t *testing.T) []asset.Model { return nil }
	plentyOfTokens := func(t *testing.T) []asset.Model {
		return []asset.Model{etcAsset(t, 1, perfectPitch, 500)}
	}

	tests := []struct {
		name            string
		etcAssets       func(t *testing.T) []asset.Model
		useCapacity     uint32
		useAssets       func(t *testing.T) []asset.Model
		tokenTemplateId uint32
		tokenPrice      uint32
		quantity        uint32
		wantError       string
	}{
		{
			name:            "insufficient tokens",
			etcAssets:       func(t *testing.T) []asset.Model { return []asset.Model{etcAsset(t, 1, perfectPitch, 4)} },
			useCapacity:     24,
			useAssets:       noAssets,
			tokenTemplateId: perfectPitch,
			tokenPrice:      5,
			quantity:        1,
			wantError:       shops.ErrorNeedMoreItems,
		},
		{
			name:            "holds none of the token item at all",
			etcAssets:       noAssets,
			useCapacity:     24,
			useAssets:       noAssets,
			tokenTemplateId: perfectPitch,
			tokenPrice:      5,
			quantity:        1,
			wantError:       shops.ErrorNeedMoreItems,
		},
		{
			name:        "destination compartment is full",
			etcAssets:   plentyOfTokens,
			useCapacity: 1,
			useAssets: func(t *testing.T) []asset.Model {
				return []asset.Model{etcAsset(t, 1, testItemId, 1)}
			},
			tokenTemplateId: perfectPitch,
			tokenPrice:      5,
			quantity:        1,
			wantError:       shops.ErrorInventoryFull,
		},
		{
			name:            "token price with no token item configured",
			etcAssets:       plentyOfTokens,
			useCapacity:     24,
			useAssets:       noAssets,
			tokenTemplateId: 0,
			tokenPrice:      5,
			quantity:        1,
			wantError:       shops.ErrorGenericError,
		},
		{
			name:            "no price configured at all",
			etcAssets:       plentyOfTokens,
			useCapacity:     24,
			useAssets:       noAssets,
			tokenTemplateId: perfectPitch,
			tokenPrice:      0,
			quantity:        1,
			wantError:       shops.ErrorGenericError,
		},
		{
			name:            "zero quantity",
			etcAssets:       plentyOfTokens,
			useCapacity:     24,
			useAssets:       noAssets,
			tokenTemplateId: perfectPitch,
			tokenPrice:      5,
			quantity:        0,
			wantError:       shops.ErrorGenericError,
		},
		{
			name:            "cost overflows uint32",
			etcAssets:       plentyOfTokens,
			useCapacity:     24,
			useAssets:       noAssets,
			tokenTemplateId: perfectPitch,
			tokenPrice:      100000,
			quantity:        100000,
			wantError:       shops.ErrorGenericError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testProcessor(t)
			buf := message.NewBuffer()
			c := testCharacter(t, tt.etcAssets(t), tt.useCapacity, tt.useAssets(t))
			cm := testCommodity(t, tt.tokenTemplateId, tt.tokenPrice)

			if err := p.buyWithTokens(buf)(c, cm, testItemId, tt.quantity); err != nil {
				t.Fatalf("buyWithTokens returned an error: %v", err)
			}

			all := buf.GetAll()

			// The strongest available form of "no partial purchase": nothing
			// at all was published on the compartment topic.
			if got := len(all[compartmentMessage.EnvCommandTopic]); got != 0 {
				t.Errorf("expected zero compartment commands on refusal, got %d", got)
			}

			events := all[shops.EnvStatusEventTopic]
			if len(events) != 1 {
				t.Fatalf("expected exactly one status event, got %d", len(events))
			}
			ev := decodeStatusError(t, events[0].Value)
			if ev.Type != shops.StatusEventTypeError {
				t.Errorf("status event type: got %q want %q", ev.Type, shops.StatusEventTypeError)
			}
			if ev.CharacterId != testCharacterId {
				t.Errorf("status event characterId: got %d want %d", ev.CharacterId, testCharacterId)
			}
			if ev.Body.Error != tt.wantError {
				t.Errorf("status event error: got %q want %q", ev.Body.Error, tt.wantError)
			}
			if ev.Body.Reason != "" {
				t.Errorf("expected no reason on a typed refusal, got %q", ev.Body.Reason)
			}
		})
	}
}
