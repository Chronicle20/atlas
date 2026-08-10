package respawn

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/compartment"
	"testing"
	"time"

	"github.com/google/uuid"

	channelInventory "atlas-channel/inventory"

	inventoryConst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
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
