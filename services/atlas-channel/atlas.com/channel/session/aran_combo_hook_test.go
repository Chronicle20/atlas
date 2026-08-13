package session

import (
	"atlas-channel/character/combo"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// This file pins Task 9's session-destroy edge for the Aran combo counter's
// mirror, mirroring battleship_hook_test.go's pattern: it calls
// clearAranComboOnDestroy directly rather than the full Destroy, which would
// require a live Kafka broker for the logout command and DESTROYED status
// event. Unlike battleship, combo.GetMirror() is not seamed behind a mockable
// constructor -- it is the real, process-wide singleton -- so these tests
// seed and read it directly, using distinct character ids and fresh tenants
// so they cannot leak state into (or race with) other tests in this package
// or in the combo package's own tests.

func aranHookTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

func aranHookTestField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
}

// TestClearAranComboOnDestroy_NonZeroCharacter_ClearsState pins the positive
// case: a destroyed session with a real character must drop that character's
// live mirror entry.
func TestClearAranComboOnDestroy_NonZeroCharacter_ClearsState(t *testing.T) {
	tn := aranHookTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)
	characterId := uint32(884201)
	now := time.Now()

	combo.GetMirror().SetEligibility(tn, characterId, aranHookTestField(), combo.NewEligibility(skill3.AranStage1ComboAbilityId, 20, 20), now)
	combo.GetMirror().Increment(tn, characterId, aranHookTestField(), combo.DefaultIdleWindow, now)

	if _, ok := combo.GetMirror().Eligibility(tn, characterId, now, 60*time.Second); !ok {
		t.Fatal("test setup invalid: want a seeded eligibility before the hook runs, got none")
	}

	clearAranComboOnDestroy(ctx, characterId)

	if _, ok := combo.GetMirror().Eligibility(tn, characterId, now, 60*time.Second); ok {
		t.Error("after clearAranComboOnDestroy: want no cached eligibility, got one")
	}
	// A cleared entry re-seeds on its next increment, confirming the count
	// itself (not just the cached eligibility) was dropped.
	count, seeded := combo.GetMirror().Increment(tn, characterId, aranHookTestField(), combo.DefaultIdleWindow, now)
	if count != 1 || !seeded {
		t.Fatalf("after clearAranComboOnDestroy the next increment must re-seed: want (1,true), got (%d,%v)", count, seeded)
	}
}

// TestClearAranComboOnDestroy_ZeroCharacter_NoOp pins the negative case: a
// session that never reached character selection (CharacterId == 0) must not
// touch the mirror at all -- in particular it must not clear character id 0
// in some other tenant's bucket.
func TestClearAranComboOnDestroy_ZeroCharacter_NoOp(t *testing.T) {
	tn := aranHookTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)
	now := time.Now()

	combo.GetMirror().SetEligibility(tn, 0, aranHookTestField(), combo.NewEligibility(skill3.AranStage1ComboAbilityId, 20, 20), now)

	clearAranComboOnDestroy(ctx, 0)

	if _, ok := combo.GetMirror().Eligibility(tn, 0, now, 60*time.Second); !ok {
		t.Error("clearAranComboOnDestroy(ctx, 0) must be a no-op: want the character-0 entry untouched, got it cleared")
	}

	// Clean up directly (bypassing the no-op hook under test) so this entry
	// cannot leak into another test in the package.
	combo.GetMirror().Clear(tn, 0)
}
