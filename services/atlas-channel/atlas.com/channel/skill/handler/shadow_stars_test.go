package handler

import (
	"atlas-channel/asset"
	"atlas-channel/data/skill/effect/statup"
	"testing"

	"github.com/google/uuid"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
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

// TestRewriteShadowClawStatups_AppendsWhenAbsent covers the production
// shape: atlas-data's produceBuffStatAmount drops a zero-value SHADOW_CLAW
// statup (its `if value != 0` guard), so the statups reaching the channel
// for skill 4121006 carry NO SHADOW_CLAW entry at all. rewriteShadowClawStatups
// must append one carrying the star id rather than silently no-op'ing, or the
// cast consumes stars while the buff never reaches the client. Mirrors
// tamedMountStatups' append-if-missing branch in mount.go.
func TestRewriteShadowClawStatups_AppendsWhenAbsent(t *testing.T) {
	in := []statup.Model{
		statup.NewModel(string(charconst.TemporaryStatTypeShadowPartner), 5),
	}
	out := rewriteShadowClawStatups(in, starIlbi)
	var sawClaw, sawPartner bool
	for _, su := range out {
		switch su.Mask() {
		case string(charconst.TemporaryStatTypeShadowClaw):
			sawClaw = true
			if su.Amount() != int32(starIlbi) {
				t.Fatalf("appended SHADOW_CLAW amount = %d, want %d", su.Amount(), starIlbi)
			}
		case string(charconst.TemporaryStatTypeShadowPartner):
			sawPartner = true
			if su.Amount() != 5 {
				t.Fatalf("non-SHADOW_CLAW statup mutated: amount = %d, want 5", su.Amount())
			}
		}
	}
	if !sawClaw {
		t.Fatalf("expected SHADOW_CLAW to be appended when absent from input; out = %+v", out)
	}
	if !sawPartner {
		t.Fatalf("expected existing statups preserved; out = %+v", out)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2 (1 preserved + 1 appended)", len(out))
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

// TestResolveShadowStarsCast_NoShadowClawInInput proves the composed path
// (validate -> consume -> rewrite) still yields a SHADOW_CLAW statup carrying
// the star id when the input statups (as atlas-data actually produces them
// for 4121006) lack a SHADOW_CLAW entry entirely.
func TestResolveShadowStarsCast_NoShadowClawInInput(t *testing.T) {
	statups := []statup.Model{statup.NewModel(string(charconst.TemporaryStatTypeShadowPartner), 5)}
	assets := []asset.Model{starAsset(1, starIlbi, 200)}

	rewritten, draws, shortfall, ok := resolveShadowStarsCast(assets, statups, starIlbi, 200)
	if !ok || shortfall {
		t.Fatalf("valid star: ok=%v shortfall=%v, want ok=true shortfall=false", ok, shortfall)
	}
	var sawClaw bool
	for _, su := range rewritten {
		if su.Mask() == string(charconst.TemporaryStatTypeShadowClaw) {
			sawClaw = true
			if su.Amount() != int32(starIlbi) {
				t.Fatalf("SHADOW_CLAW amount = %d, want %d", su.Amount(), starIlbi)
			}
		}
	}
	if !sawClaw {
		t.Fatalf("expected SHADOW_CLAW appended to rewritten statups when absent from input; rewritten = %+v", rewritten)
	}
	total := 0
	for _, d := range draws {
		total += int(d.Quantity)
	}
	if total != 200 {
		t.Fatalf("drawn total = %d, want 200", total)
	}
}
