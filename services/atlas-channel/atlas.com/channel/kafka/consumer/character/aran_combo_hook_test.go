package character

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

// This file pins the map-changed reset hook for the Aran combo counter's
// mirror (task-217 design.md §3.4). clearAranComboOnMapChange was extracted
// from handleStatusEventMapChanged precisely so it is unit-testable without
// exercising the consumer's full event path -- that path needs a
// server.Model (unexported fields, built only inside the server package) and
// a registered session. combo.GetMirror() is the real, process-wide
// singleton, so these tests seed and read it directly, using a distinct
// character id and a fresh tenant so they cannot leak state into (or race
// with) other tests in this package or elsewhere.

func aranMapChangeTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

func aranMapChangeTestField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
}

// TestClearAranComboOnMapChange_ClearsState pins the positive case: a
// character that just changed maps must have its live mirror entry dropped.
func TestClearAranComboOnMapChange_ClearsState(t *testing.T) {
	tn := aranMapChangeTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)
	characterId := uint32(773105)
	now := time.Now()

	combo.GetMirror().SetEligibility(tn, characterId, aranMapChangeTestField(), combo.NewEligibility(skill3.AranStage1ComboAbilityId, 20, 20), now)
	combo.GetMirror().Increment(tn, characterId, aranMapChangeTestField(), combo.DefaultIdleWindow, now)

	if _, ok := combo.GetMirror().Eligibility(tn, characterId, now, 60*time.Second); !ok {
		t.Fatal("test setup invalid: want a seeded eligibility before the hook runs, got none")
	}

	clearAranComboOnMapChange(ctx, characterId)

	if _, ok := combo.GetMirror().Eligibility(tn, characterId, now, 60*time.Second); ok {
		t.Error("after clearAranComboOnMapChange: want no cached eligibility, got one")
	}
	// A cleared entry re-seeds on its next increment, confirming the count
	// itself (not just the cached eligibility) was dropped.
	count, seeded := combo.GetMirror().Increment(tn, characterId, aranMapChangeTestField(), combo.DefaultIdleWindow, now)
	if count != 1 || !seeded {
		t.Fatalf("after clearAranComboOnMapChange the next increment must re-seed: want (1,true), got (%d,%v)", count, seeded)
	}
}

// TestClearAranComboOnMapChange_UnknownCharacter_NoOp pins the negative case:
// clearing a character with no live entry must not panic or affect other
// characters' state in the same tenant bucket.
func TestClearAranComboOnMapChange_UnknownCharacter_NoOp(t *testing.T) {
	tn := aranMapChangeTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)
	otherCharacterId := uint32(773106)
	unknownCharacterId := uint32(773107)
	now := time.Now()

	combo.GetMirror().SetEligibility(tn, otherCharacterId, aranMapChangeTestField(), combo.NewEligibility(skill3.AranStage1ComboAbilityId, 20, 20), now)

	clearAranComboOnMapChange(ctx, unknownCharacterId)

	if _, ok := combo.GetMirror().Eligibility(tn, otherCharacterId, now, 60*time.Second); !ok {
		t.Error("clearAranComboOnMapChange for an unrelated character must not touch another character's entry")
	}

	combo.GetMirror().Clear(tn, otherCharacterId)
}
