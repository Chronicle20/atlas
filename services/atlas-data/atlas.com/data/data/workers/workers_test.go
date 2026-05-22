package workers

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestRegisteredSize(t *testing.T) {
	if len(Registered) != 11 {
		t.Fatalf("registered = %d, want 11", len(Registered))
	}
}

func TestRegisteredUniqueArchives(t *testing.T) {
	seen := map[string]bool{}
	for _, w := range Registered {
		if seen[w.ArchiveName()] {
			t.Fatalf("duplicate archive: %s", w.ArchiveName())
		}
		seen[w.ArchiveName()] = true
	}
}

func TestRegisteredUniqueNames(t *testing.T) {
	seen := map[string]bool{}
	for _, w := range Registered {
		if seen[w.Name()] {
			t.Fatalf("duplicate name: %s", w.Name())
		}
		seen[w.Name()] = true
	}
}

// TestRegisteredCoversExpectedDomains locks in the worker inventory the new
// MinIO-backed pipeline ships with. Removing or renaming a worker is a
// deliberate code change that must also touch this list — that's the point.
//
// Why this exists: COMMODITY was missing for the entire task-071 cycle
// (8,941 atlas-data documents absent from PR-544 vs main) because nothing
// enforced the inventory. Adding the legacy MONSTER worker (now folded into
// MOB), PET / CONSUME / CASH / ETC / SETUP (folded into ITEM), or FACE /
// HAIR / CHARACTER_CREATION (folded into CHARACTER), or MOB_SKILL (folded
// into SKILL) requires updating both the worker registry AND the comment
// below, so the "umbrella" boundary stays auditable.
func TestRegisteredCoversExpectedDomains(t *testing.T) {
	expected := map[string]string{
		// name -> short comment of what the worker covers
		"MAP":       "Map.wz: maps + spawn indexes + per-map render assets",
		"MOB":       "Mob.wz: monsters (was MONSTER) + mob icons",
		"NPC":       "Npc.wz: NPCs + npc icons",
		"REACTOR":   "Reactor.wz: reactors + reactor icons",
		"SKILL":     "Skill.wz: skills + skill icons + MOB_SKILL (folded)",
		"QUEST":     "Quest.wz: quests",
		"STRING":    "String.wz: string registries",
		"CHARACTER": "Character.wz: equipment + FACE + HAIR + CHARACTER_CREATION (folded) + character atlases",
		"UI":        "UI.wz: world icons + gauge metadata",
		"ITEM":      "Item.wz: CONSUME + CASH + ETC + SETUP + PET (folded) + item icons",
		"COMMODITY": "Etc.wz/Commodity.img.xml: cash-shop commodities",
	}
	present := map[string]bool{}
	for _, w := range Registered {
		present[w.Name()] = true
	}
	for name := range expected {
		if !present[name] {
			t.Errorf("Registered missing worker: %s (%s)", name, expected[name])
		}
	}
	for n := range present {
		if _, ok := expected[n]; !ok {
			t.Errorf("Registered has unexpected worker %q — add to expected map or remove from Registered", n)
		}
	}
}

// TestWithTenantPreInjection locks in the dispatcher contract that
// data.RunWorkers MUST establish before invoking any Worker.Run: the
// context passed to Run carries the tenant from Params, so downstream
// tenant.MustFromContext calls don't panic regardless of how (or whether)
// individual workers also call withTenant. This guards against a recurrence
// of the Commodity panic from f247e976f.
func TestWithTenantPreInjection(t *testing.T) {
	p := Params{
		ScopeKey:     "tenants/00000000-0000-0000-0000-000000000001",
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
	}
	ctx, model, err := WithTenant(context.Background(), p)
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
	if model.Id().String() != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("tenant id = %s, want from ScopeKey", model.Id())
	}
	// Round-trip: the resulting ctx MUST satisfy tenant.MustFromContext
	// without panicking. If WithTenant ever stops injecting (or starts
	// injecting against a key MustFromContext doesn't read), this test
	// surfaces it immediately rather than letting workers crash in prod.
	defer func() {
		if r := recover(); r != nil {
			if isTenantPanic(r, "retrieve id from context") {
				t.Fatalf("WithTenant did not put tenant where MustFromContext reads it (panic: %v)", r)
			}
			panic(r)
		}
	}()
	got := tenant.MustFromContext(ctx)
	if got.Id() != model.Id() {
		t.Fatalf("round-trip tenant id mismatch: %s vs %s", got.Id(), model.Id())
	}
}

func isTenantPanic(r interface{}, substr string) bool {
	switch v := r.(type) {
	case string:
		return strings.Contains(v, substr)
	case error:
		return strings.Contains(v.Error(), substr)
	default:
		return strings.Contains(fmt.Sprintf("%v", r), substr)
	}
}
