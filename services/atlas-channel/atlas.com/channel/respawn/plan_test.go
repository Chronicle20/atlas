package respawn

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/compartment"
	"atlas-channel/saga"
	"testing"
	"time"

	"github.com/google/uuid"

	channelInventory "atlas-channel/inventory"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventoryConst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	characterId = uint32(4096)
	currentMap  = _map.Id(104040000)
	returnMap   = _map.Id(104000000)
)

// ordinaryField is a non-town map with no field limits and a distinct return
// map — the plain case every test varies from.
var ordinaryField = mapFacts{ReturnMapId: returnMap, Town: false, NoExpLossOnDeath: false}

func buildCharacter(experience uint32) character.Model {
	return character.NewModelBuilder().
		SetId(characterId).
		SetName("Tumi").
		SetLevel(30).
		SetJobId(job.WarriorId).
		SetLuck(60).
		SetExperience(experience).
		SetHp(0).
		SetMaxHp(1000).
		MustBuild()
}

// buildAsset creates one stack of templateId with the given quantity and
// expiration (pass the zero time for "no expiration").
func buildAsset(compartmentId uuid.UUID, templateId uint32, quantity uint32, expiration time.Time) asset.Model {
	return asset.NewModelBuilder(1, compartmentId, templateId).
		SetSlot(1).
		SetQuantity(quantity).
		SetExpiration(expiration).
		MustBuild()
}

// buildInventory places each (templateId, quantity) pair into the compartment
// its inventory type dictates: cash items in Cash, the ETC protection items in
// ETC. A quantity of 0 means "the asset is absent entirely".
func buildInventory(items map[uint32]uint32) channelInventory.Model {
	cashId, etcId := uuid.New(), uuid.New()
	cash := compartment.NewModelBuilder(cashId, characterId, inventoryConst.TypeValueCash, 100)
	etc := compartment.NewModelBuilder(etcId, characterId, inventoryConst.TypeValueETC, 100)
	for templateId, quantity := range items {
		if quantity == 0 {
			continue
		}
		if item.Id(templateId) == item.EasterBasketId || item.Id(templateId) == item.ProtectOnDeathId {
			etc = etc.AddAsset(buildAsset(etcId, templateId, quantity, time.Time{}))
			continue
		}
		cash = cash.AddAsset(buildAsset(cashId, templateId, quantity, time.Time{}))
	}
	return channelInventory.NewModelBuilder(characterId).
		SetCash(cash.MustBuild()).
		SetEtc(etc.MustBuild()).
		MustBuild()
}

func TestPlanRespawn_PremiumZeroKeepsTheWheel(t *testing.T) {
	inv := buildInventory(map[uint32]uint32{uint32(item.WheelOfFortuneId): 1})

	got := planRespawn(buildCharacter(0), inv, ordinaryField, currentMap, false)

	if got.TargetMapId != returnMap {
		t.Errorf("target map: got %d, want %d (return map — the player pressed Cancel)", got.TargetMapId, returnMap)
	}
	if got.Wheel != nil {
		t.Error("wheel must not be consumed when the client sent premium = 0")
	}
}

func TestPlanRespawn_PremiumOneWithWheelStaysInMap(t *testing.T) {
	inv := buildInventory(map[uint32]uint32{uint32(item.WheelOfFortuneId): 1})

	got := planRespawn(buildCharacter(0), inv, ordinaryField, currentMap, true)

	if got.TargetMapId != currentMap {
		t.Errorf("target map: got %d, want %d (in-map revive)", got.TargetMapId, currentMap)
	}
	if got.Wheel == nil {
		t.Fatal("wheel should be consumed on an in-map revive")
	}
	if got.Wheel.Quantity() != 1 {
		t.Errorf("wheel quantity: got %d, want 1", got.Wheel.Quantity())
	}
}

func TestPlanRespawn_PremiumOneWithoutWheelUsesReturnMap(t *testing.T) {
	inv := buildInventory(map[uint32]uint32{})

	got := planRespawn(buildCharacter(0), inv, ordinaryField, currentMap, true)

	if got.TargetMapId != returnMap {
		t.Errorf("target map: got %d, want %d", got.TargetMapId, returnMap)
	}
	if got.Wheel != nil {
		t.Error("no wheel owned, nothing to consume")
	}
}

func TestPlanRespawn_ZeroQuantityWheelIsNotUsable(t *testing.T) {
	// A wheel present at quantity 0 must read exactly like an absent one.
	cashId := uuid.New()
	cash := compartment.NewModelBuilder(cashId, characterId, inventoryConst.TypeValueCash, 100).
		AddAsset(buildAsset(cashId, uint32(item.WheelOfFortuneId), 0, time.Time{})).
		MustBuild()
	inv := channelInventory.NewModelBuilder(characterId).SetCash(cash).MustBuild()

	got := planRespawn(buildCharacter(0), inv, ordinaryField, currentMap, true)
	if got.Wheel != nil || got.TargetMapId != returnMap {
		t.Errorf("a wheel with no charges must not redirect the respawn: %+v", got)
	}
}

func TestUsesRemaining(t *testing.T) {
	tests := []struct {
		name     string
		quantity uint32
		nilAsset bool
		want     byte
	}{
		{name: "nil asset", nilAsset: true, want: 0},
		{name: "last charge", quantity: 1, want: 0},
		{name: "three charges", quantity: 3, want: 2},
		{name: "clamped at 255", quantity: 1000, want: 255},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var a *asset.Model
			if !tc.nilAsset {
				built := buildAsset(uuid.New(), uint32(item.WheelOfFortuneId), tc.quantity, time.Time{})
				a = &built
			}
			if got := usesRemaining(a); got != tc.want {
				t.Errorf("usesRemaining(quantity %d) = %d, want %d", tc.quantity, got, tc.want)
			}
		})
	}
}

func TestPlanRespawn_MultipleChargesDecrementRatherThanDestroy(t *testing.T) {
	inv := buildInventory(map[uint32]uint32{uint32(item.WheelOfFortuneId): 3})

	got := planRespawn(buildCharacter(0), inv, ordinaryField, currentMap, true)
	if got.Wheel == nil {
		t.Fatal("expected the wheel to be selected")
	}
	if usesRemaining(got.Wheel) != 2 {
		t.Errorf("post-decrement charges: got %d, want 2", usesRemaining(got.Wheel))
	}
	// The saga step removes exactly one unit and never the whole stack, so a
	// quantity-3 wheel survives and a quantity-1 wheel is destroyed by the
	// same step. Task 6's createRespawnSaga always passes
	// Quantity: 1, RemoveAll: false — pinned by TestRespawnSagaStepOrdering.
}

func stepIds(steps []saga.Step) []string {
	ids := make([]string, 0, len(steps))
	for _, s := range steps {
		ids = append(ids, s.StepId)
	}
	return ids
}

// Both consume steps must precede warp_to_spawn: the saga stops at the first
// failing step, so a failed decrement can never leave the player revived for
// free in the map they died in.
func TestRespawnSagaStepOrdering(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), currentMap).Build()
	inv := buildInventory(map[uint32]uint32{uint32(item.WheelOfFortuneId): 3})
	rp := planRespawn(buildCharacter(0), inv, ordinaryField, currentMap, true)

	steps := respawnSagaSteps(f, characterId, rp, time.Now())

	ids := stepIds(steps)
	consume, warp := -1, -1
	for i, id := range ids {
		switch id {
		case "consume_wheel_of_fortune":
			consume = i
		case "warp_to_spawn":
			warp = i
		}
	}
	if consume == -1 || warp == -1 {
		t.Fatalf("expected both a consume and a warp step, got %v", ids)
	}
	if consume > warp {
		t.Errorf("consume must precede warp, got %v", ids)
	}
	payload, ok := steps[consume].Payload.(saga.DestroyAssetPayload)
	if !ok {
		t.Fatalf("consume payload is %T, want saga.DestroyAssetPayload", steps[consume].Payload)
	}
	if payload.Quantity != 1 || payload.RemoveAll {
		t.Errorf("consume payload: got quantity %d removeAll %v, want 1/false", payload.Quantity, payload.RemoveAll)
	}
	if payload.TemplateId != uint32(item.WheelOfFortuneId) {
		t.Errorf("consume template: got %d, want %d", payload.TemplateId, item.WheelOfFortuneId)
	}
}

// FR-2.3: one death consumes exactly one charge even when the client sends
// both USE_DEATHITEM and MAP_CHANGE. USE_DEATHITEM is handled by
// CharacterUseDeathItemHandleFunc, which builds no saga and consumes nothing,
// so the only DestroyAsset for the wheel is the one MAP_CHANGE produces here.
func TestOneDeathConsumesOneCharge(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), currentMap).Build()
	inv := buildInventory(map[uint32]uint32{uint32(item.WheelOfFortuneId): 3})
	rp := planRespawn(buildCharacter(0), inv, ordinaryField, currentMap, true)

	steps := respawnSagaSteps(f, characterId, rp, time.Now())

	wheelConsumes := 0
	for _, s := range steps {
		if p, ok := s.Payload.(saga.DestroyAssetPayload); ok && p.TemplateId == uint32(item.WheelOfFortuneId) {
			wheelConsumes++
		}
	}
	if wheelConsumes != 1 {
		t.Errorf("wheel DestroyAsset steps: got %d, want exactly 1", wheelConsumes)
	}
}
