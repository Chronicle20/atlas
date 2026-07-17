package handler

import (
	"testing"

	"atlas-channel/asset"
	"atlas-channel/data/skill/effect/statup"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/google/uuid"
)

const (
	starIlbi uint32 = 2070006 // Ilbi Throwing Stars (classification 207)
	starSubi uint32 = 2070000 // Subi Throwing Stars (classification 207)
	notAStar uint32 = 2000000 // Red Potion (classification 200)
)

func starAsset(slot int16, templateId uint32, qty uint32) asset.Model {
	return asset.NewModelBuilder(1, uuid.New(), templateId).
		SetSlot(slot).
		SetQuantity(qty).
		MustBuild()
}

func TestValidateShadowStar(t *testing.T) {
	assets := []asset.Model{starAsset(1, starIlbi, 50)}
	if !validateShadowStar(assets, starIlbi) {
		t.Fatalf("owned throwing star should validate")
	}
	if validateShadowStar(assets, starSubi) {
		t.Fatalf("unowned throwing star should not validate")
	}
	if validateShadowStar(assets, notAStar) {
		t.Fatalf("non-throwing-star id should not validate")
	}
	if validateShadowStar([]asset.Model{starAsset(1, starIlbi, 0)}, starIlbi) {
		t.Fatalf("zero-quantity star should not validate")
	}
}

func TestResolveStarConsume_SingleSlot(t *testing.T) {
	assets := []asset.Model{starAsset(1, starIlbi, 200), starAsset(2, starSubi, 200)}
	draws, available := resolveStarConsume(assets, starIlbi, 200)
	if available != 200 {
		t.Fatalf("available = %d, want 200", available)
	}
	if len(draws) != 1 || draws[0].Slot != 1 || draws[0].ItemId != starIlbi || draws[0].Quantity != 200 {
		t.Fatalf("draws = %+v, want single slot-1 draw of 200 Ilbi", draws)
	}
}

func TestResolveStarConsume_MultiSlotAndShortfall(t *testing.T) {
	// 120 across two Ilbi slots; a Subi slot must be ignored.
	assets := []asset.Model{starAsset(1, starIlbi, 80), starAsset(2, starSubi, 200), starAsset(3, starIlbi, 40)}
	draws, available := resolveStarConsume(assets, starIlbi, 200)
	if available != 120 {
		t.Fatalf("available = %d, want 120 (shortfall)", available)
	}
	total := 0
	for _, d := range draws {
		if d.ItemId != starIlbi {
			t.Fatalf("draw targeted wrong item %d, want %d", d.ItemId, starIlbi)
		}
		total += int(d.Quantity)
	}
	if total != 120 {
		t.Fatalf("drawn total = %d, want 120", total)
	}
}

func TestRewriteShadowClawStatups(t *testing.T) {
	in := []statup.Model{
		statup.NewModel(string(charconst.TemporaryStatTypeShadowClaw), 0),
		statup.NewModel(string(charconst.TemporaryStatTypeShadowPartner), 5),
	}
	out := rewriteShadowClawStatups(in, starIlbi)
	var sawClaw, sawPartner bool
	for _, su := range out {
		switch su.Mask() {
		case string(charconst.TemporaryStatTypeShadowClaw):
			sawClaw = true
			if su.Amount() != int32(starIlbi) {
				t.Fatalf("SHADOW_CLAW amount = %d, want %d", su.Amount(), starIlbi)
			}
		case string(charconst.TemporaryStatTypeShadowPartner):
			sawPartner = true
			if su.Amount() != 5 {
				t.Fatalf("non-SHADOW_CLAW statup mutated: amount = %d, want 5", su.Amount())
			}
		}
	}
	if !sawClaw || !sawPartner {
		t.Fatalf("expected both statups preserved; sawClaw=%v sawPartner=%v", sawClaw, sawPartner)
	}
}

func TestResolveShadowStarsCast(t *testing.T) {
	statups := []statup.Model{statup.NewModel(string(charconst.TemporaryStatTypeShadowClaw), 0)}

	// Invalid star -> abort, no draws, no rewrite.
	if _, draws, _, ok := resolveShadowStarsCast(nil, statups, starIlbi, 200); ok || len(draws) != 0 {
		t.Fatalf("unowned star: ok=%v draws=%d, want ok=false and no draws", ok, len(draws))
	}

	// Valid star -> SHADOW_CLAW carries star id, draws total bulletCount, no shortfall.
	assets := []asset.Model{starAsset(1, starIlbi, 200)}
	rewritten, draws, shortfall, ok := resolveShadowStarsCast(assets, statups, starIlbi, 200)
	if !ok || shortfall {
		t.Fatalf("valid star: ok=%v shortfall=%v, want ok=true shortfall=false", ok, shortfall)
	}
	if len(rewritten) != 1 || rewritten[0].Amount() != int32(starIlbi) {
		t.Fatalf("rewritten SHADOW_CLAW amount = %+v, want %d", rewritten, starIlbi)
	}
	total := 0
	for _, d := range draws {
		total += int(d.Quantity)
	}
	if total != 200 {
		t.Fatalf("drawn total = %d, want 200", total)
	}
}
