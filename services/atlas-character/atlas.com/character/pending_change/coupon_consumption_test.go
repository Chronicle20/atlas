package pending_change

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TestApplyConsumesTheNameChangeCoupons pins that a landed rename takes the
// coupons with it.
//
// The action matters as much as the count: it must be destroy_all_assets, not
// destroy_asset. Cash items do not stack — each instance carries its own cashId
// and occupies its own slot — and destroy_asset resolves a template to the FIRST
// matching slot only, so a player holding two coupons would keep one. A version
// of this emit that used destroy_asset would pass a bare "one command was
// emitted" assertion, which is why this test names the action.
func TestApplyConsumesTheNameChangeCoupons(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	characterId := seedCharacter(t, db, "Uniform", world.Id(0))

	p := NewProcessor(l, ctx, db)
	// assetId nil: the purchase path, which is the only one atlas-channel
	// produces. The coupon is materialised by the cash-shop purchase AFTER the
	// request, so consumption cannot happen at request acceptance.
	if _, err := p.CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Victor", world.Id(0), nil); err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}
	if got := countOutboxMessagesMatching(t, db, "consume_name_change_coupons"); got != 0 {
		t.Fatalf("no coupon consumption may be emitted at request time, got %d", got)
	}

	if err := p.ApplyForCharacter(characterId); err != nil {
		t.Fatalf("ApplyForCharacter: %v", err)
	}

	if got := countOutboxMessagesMatching(t, db, "consume_name_change_coupons"); got != 1 {
		t.Fatalf("expected one coupon-consumption command on apply, got %d", got)
	}
	if got := countOutboxMessagesMatching(t, db, "destroy_all_assets"); got != 1 {
		t.Fatalf("consumption must use destroy_all_assets (destroy_asset clears only the first slot), got %d", got)
	}
	if got := countOutboxMessagesMatching(t, db, "5400000"); got != 1 {
		t.Fatalf("expected the consumption to name the name-change coupon template, got %d", got)
	}
}

// A rejected apply (the name was taken in the interim) must NOT eat the
// coupons — the player still holds an unspent change.
func TestRejectedApplyLeavesTheCouponsAlone(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	characterId := seedCharacter(t, db, "Whiskey", world.Id(0))
	p := NewProcessor(l, ctx, db)

	if _, err := p.CreateAndEmit(uuid.New(), characterId, TypeNameChange, "Xray", world.Id(0), nil); err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}
	// Someone else takes the name before the character logs out.
	seedCharacter(t, db, "Xray", world.Id(0))

	if err := p.ApplyForCharacter(characterId); err != nil {
		t.Fatalf("ApplyForCharacter: %v", err)
	}

	if got := countOutboxMessagesMatching(t, db, "consume_name_change_coupons"); got != 0 {
		t.Fatalf("a rejected apply must not consume coupons, got %d", got)
	}
}
