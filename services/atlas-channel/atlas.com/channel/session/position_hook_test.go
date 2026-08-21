package session

import (
	"atlas-channel/position"
	"context"
	"testing"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// This file pins task-250's session-destroy edge for the last-known-position
// registry's mirror, mirroring aran_combo_hook_test.go's pattern: it calls
// clearLastPositionOnDestroy directly rather than the full Destroy, which
// would require a live Kafka broker for the logout command and DESTROYED
// status event. Unlike battleship, position.GetRegistry() is not seamed
// behind a mockable constructor -- it is the real, process-wide singleton --
// so these tests seed and read it directly, using distinct character ids and
// fresh tenants so they cannot leak state into (or race with) other tests in
// this package or in the position package's own tests.

func positionHookTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

// TestClearLastPositionOnDestroy_NonZeroCharacter_ClearsState pins the
// positive case: a destroyed session with a real character must drop that
// character's live last-known-position entry.
func TestClearLastPositionOnDestroy_NonZeroCharacter_ClearsState(t *testing.T) {
	tn := positionHookTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)
	characterId := uint32(884202)

	position.GetRegistry().Put(tn, characterId, position.Position{X: 100, Y: 200})

	if _, ok := position.GetRegistry().Lookup(tn, characterId); !ok {
		t.Fatal("test setup invalid: want a seeded position before the hook runs, got none")
	}

	clearLastPositionOnDestroy(ctx, characterId)

	if _, ok := position.GetRegistry().Lookup(tn, characterId); ok {
		t.Error("after clearLastPositionOnDestroy: want no cached position, got one")
	}
}

// TestClearLastPositionOnDestroy_ZeroCharacter_NoOp pins the negative case: a
// session that never reached character selection (CharacterId == 0) must not
// touch the registry at all -- in particular it must not clear character id 0
// in some other tenant's bucket, and must not disturb an unrelated
// character's entry.
func TestClearLastPositionOnDestroy_ZeroCharacter_NoOp(t *testing.T) {
	tn := positionHookTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)
	characterId := uint32(42)

	position.GetRegistry().Put(tn, characterId, position.Position{X: 1, Y: 2})

	clearLastPositionOnDestroy(ctx, 0)

	if _, ok := position.GetRegistry().Lookup(tn, characterId); !ok {
		t.Error("clearLastPositionOnDestroy(ctx, 0) must be a no-op: want character 42's entry untouched, got it cleared")
	}

	// Clean up directly (bypassing the no-op hook under test) so this entry
	// cannot leak into another test in the package.
	position.GetRegistry().Clear(tn, characterId)
}
