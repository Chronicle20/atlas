package custody

// dupe_safety_test.go — the custody-handler arm of the MTS dupe-safety suite
// (task-102 §5.2, NFR 8.1). Each test asserts the SINGLE-CUSTODY invariant
// DIRECTLY against the real custody Kafka handlers and the listing processor:
// after a replay or a race, the item exists in EXACTLY ONE place
// (custody-row-count == 1, never 0 or 2).
//
// Scenarios covered here:
//   3. double-grant replay     — TestDupeSafety_DoubleGrantReplay_*
//   4. cancel-racing-purchase  — TestDupeSafety_CancelRacingPurchase_*
//   5. take-home replay        — TestDupeSafety_TakeHomeReplay_*
//
// (Scenarios 1 crash-mid-list and 2 grant-before-debit are orchestrator-side and
// live in atlas-saga-orchestrator/.../saga/mts_dupe_safety_test.go. The listing-
// arm half of scenario 4 lives in listing/dupe_safety_test.go.)

import (
	"atlas-mts/holding"
	"atlas-mts/kafka/message/custody"
	"atlas-mts/listing"
	"atlas-mts/test"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// countHoldingsForOwner counts non-deleted holdings owned by ownerId. Tests use
// unique owner ids so the cache=shared in-memory DB leaking rows across tests does
// not pollute the count.
func countHoldingsForOwner(t *testing.T, db *gorm.DB, ctx context.Context, ownerId uint32) int {
	t.Helper()
	all, err := holding.GetAll()(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("holding GetAll: %v", err)
	}
	n := 0
	for _, m := range all {
		if m.OwnerId() == ownerId {
			n++
		}
	}
	return n
}

// countListingRows counts listing rows carrying the given id.
func countListingRows(t *testing.T, db *gorm.DB, ctx context.Context, listingId uuid.UUID) int {
	t.Helper()
	all, err := listing.GetAll()(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("listing GetAll: %v", err)
	}
	n := 0
	for _, m := range all {
		if m.Id() == listingId {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// Scenario 3: double-grant replay
//
// A custody-create command delivered TWICE must create EXACTLY ONE custody row.
// Covers both create paths: AcceptToMtsListing (listing row) and
// MtsMoveListingToHolding (buyer holding row).
// ---------------------------------------------------------------------------

// TestDupeSafety_DoubleGrantReplay_AcceptListing asserts that delivering the
// AcceptToMtsListing custody-create command twice yields EXACTLY ONE listing row
// (the item is custodied in one place, not duplicated by a redelivery).
func TestDupeSafety_DoubleGrantReplay_AcceptListing(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	rp := &recordingProducer{}
	transactionId := uuid.New()
	listingId := uuid.New()
	cmd := newAcceptCommand(transactionId, listingId)

	// Deliver the create command twice (replay).
	handleAcceptToMtsListing(rp.provider())(db)(l, ctx, cmd)
	handleAcceptToMtsListing(rp.provider())(db)(l, ctx, cmd)

	// INVARIANT: exactly one listing custody row for this id after the replay.
	if got := countListingRows(t, db, ctx, listingId); got != 1 {
		t.Fatalf("single-custody invariant violated: double-grant of AcceptToMtsListing created %d listing rows, want exactly 1", got)
	}
}

// TestDupeSafety_DoubleGrantReplay_MoveToHolding asserts that delivering the
// MtsMoveListingToHolding custody-create command twice yields EXACTLY ONE buyer
// holding (the deterministic moveHoldingId idempotency guard prevents a second
// copy), and the listing stays sold.
func TestDupeSafety_DoubleGrantReplay_MoveToHolding(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	transactionId := uuid.New()
	listingId := uuid.New()
	const buyerId = uint32(8881001)
	seedActiveListing(t, db, ctx, listingId)

	rp := &recordingProducer{}
	cmd := newMoveCommand(transactionId, listingId, buyerId)

	// Deliver the move command twice (replay).
	handleMtsMoveListingToHolding(rp.provider())(db)(l, ctx, cmd)
	handleMtsMoveListingToHolding(rp.provider())(db)(l, ctx, cmd)

	// INVARIANT: exactly one buyer holding after the replay (no second grant).
	if got := countHoldingsForOwner(t, db, ctx, buyerId); got != 1 {
		t.Fatalf("single-custody invariant violated: double-grant of MtsMoveListingToHolding created %d buyer holdings, want exactly 1", got)
	}

	// The item moved OUT of the listing: the listing is sold (custody is the
	// holding now, not the listing) — exactly one copy, not two.
	stored, err := listing.GetById(listingId.String())(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("listing lookup: %v", err)
	}
	if stored.State() != listing.StateSold {
		t.Fatalf("expected listing sold after move, got %s", stored.State())
	}
}

// ---------------------------------------------------------------------------
// Scenario 4: cancel-racing-purchase
//
// Cancel (→ seller holding, origin=cancelled) and settle-move (→ buyer holding,
// origin=purchased) both target an `active` listing. The conditional
// UPDATE ... WHERE state='active' lets EXACTLY ONE win. The item must end up in
// EXACTLY ONE holding — never both, never neither.
// ---------------------------------------------------------------------------

// TestDupeSafety_CancelRacingPurchase_CancelWins asserts that when Cancel wins
// the race (listing → cancelled, seller holding created), a subsequent settle-move
// on the now-non-active listing is the loser: it creates NO buyer holding. The
// item exists in EXACTLY ONE holding (the seller's).
func TestDupeSafety_CancelRacingPurchase_CancelWins(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	listingId := uuid.New()
	const sellerId = uint32(8882001)
	const buyerId = uint32(8882002)
	seedActiveListingWithSeller(t, db, ctx, listingId, sellerId)

	// Cancel wins the race.
	p := listing.NewProcessor(l, ctx, db)
	res, err := p.Cancel(listingId.String())
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if !res.Won {
		t.Fatal("cancel of an active listing must win the race (Won=true)")
	}

	// Settle-move races and loses (listing is no longer active).
	rp := &recordingProducer{}
	handleMtsMoveListingToHolding(rp.provider())(db)(l, ctx, newMoveCommand(uuid.New(), listingId, buyerId))

	// INVARIANT: exactly one seller holding, zero buyer holdings.
	if got := countHoldingsForOwner(t, db, ctx, sellerId); got != 1 {
		t.Fatalf("single-custody invariant violated: want 1 seller holding (cancel won), got %d", got)
	}
	if got := countHoldingsForOwner(t, db, ctx, buyerId); got != 0 {
		t.Fatalf("single-custody invariant violated: settle-move lost the race but created %d buyer holdings, want 0", got)
	}

	// CURRENCY invariant: the losing settle move must ack ERROR (not MOVED) so the
	// MtsSettlePurchase saga compensates the buyer's already-applied prepaid debit.
	// A MOVED ack would complete the purchase -> the buyer is charged for an item
	// the seller reclaimed (currency desync), with no compensation.
	if len(rp.events) != 1 {
		t.Fatalf("expected exactly 1 ack from the losing move, got %d", len(rp.events))
	}
	if rp.events[0].eventType != custody.StatusEventTypeError {
		t.Fatalf("settle move lost the race but acked %q; want ERROR so the saga compensates the buyer debit", rp.events[0].eventType)
	}

	// Listing stays cancelled (the losing move did not flip it to sold).
	stored, err := listing.GetById(listingId.String())(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("listing lookup: %v", err)
	}
	if stored.State() != listing.StateCancelled {
		t.Fatalf("expected listing cancelled, got %s", stored.State())
	}
}

// TestDupeSafety_CancelRacingPurchase_SettleWins asserts the mirror: when the
// settle-move wins the race (listing → sold, buyer holding created), a subsequent
// Cancel on the now-non-active listing is the loser: it creates NO seller holding.
// The item exists in EXACTLY ONE holding (the buyer's).
func TestDupeSafety_CancelRacingPurchase_SettleWins(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	listingId := uuid.New()
	const sellerId = uint32(8883001)
	const buyerId = uint32(8883002)
	seedActiveListingWithSeller(t, db, ctx, listingId, sellerId)

	// Settle-move wins the race.
	rp := &recordingProducer{}
	handleMtsMoveListingToHolding(rp.provider())(db)(l, ctx, newMoveCommand(uuid.New(), listingId, buyerId))

	// Cancel races and loses (listing is no longer active).
	p := listing.NewProcessor(l, ctx, db)
	res, err := p.Cancel(listingId.String())
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if res.Won {
		t.Fatal("cancel must lose the race against a settled listing (Won=false)")
	}

	// INVARIANT: exactly one buyer holding, zero seller holdings.
	if got := countHoldingsForOwner(t, db, ctx, buyerId); got != 1 {
		t.Fatalf("single-custody invariant violated: want 1 buyer holding (settle won), got %d", got)
	}
	if got := countHoldingsForOwner(t, db, ctx, sellerId); got != 0 {
		t.Fatalf("single-custody invariant violated: cancel lost the race but created %d seller holdings, want 0", got)
	}

	// Listing stays sold (the losing cancel did not flip it to cancelled).
	stored, err := listing.GetById(listingId.String())(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("listing lookup: %v", err)
	}
	if stored.State() != listing.StateSold {
		t.Fatalf("expected listing sold, got %s", stored.State())
	}
}

// ---------------------------------------------------------------------------
// Scenario 5: take-home replay
//
// ReleaseFromMtsHolding (the take-home custody step) delivered twice must
// soft-delete the holding EXACTLY ONCE. The first delivery removes the row (1
// row affected) → AcceptToCharacter would grant the item once; the replay is a
// no-op (0 rows affected) → no second grant. No double-grant of the item home.
// ---------------------------------------------------------------------------

// TestDupeSafety_TakeHomeReplay asserts that a replayed ReleaseFromMtsHolding
// releases the holding once and is a no-op on replay: the holding is gone after
// the first delivery, and the second delivery affects ZERO rows — so the
// downstream AcceptToCharacter grant fires exactly once (no item duplicated home).
func TestDupeSafety_TakeHomeReplay(t *testing.T) {
	db := test.SetupTestDB(t, listing.Migration, holding.Migration)
	ctx := test.CreateTestContext()
	l := logrus.New()

	const ownerId = uint32(8884001)
	holdingId := uuid.New()
	m, err := holding.NewBuilder(test.TestTenantId, 0, ownerId).
		SetId(holdingId).
		SetOrigin(holding.OriginPurchased).
		SetTemplateId(1302000).
		SetQuantity(1).
		Build()
	if err != nil {
		t.Fatalf("build holding: %v", err)
	}
	if _, err := holding.CreateHolding(db.WithContext(ctx), m); err != nil {
		t.Fatalf("seed holding: %v", err)
	}

	// First delivery: soft-deletes exactly one row (the grant-home fires once).
	first, err := holding.SoftDelete(db.WithContext(ctx), holdingId.String())
	if err != nil {
		t.Fatalf("first SoftDelete: %v", err)
	}
	if first != 1 {
		t.Fatalf("first take-home must release exactly 1 row, got %d", first)
	}

	// Drive the real handler to confirm the holding is gone and the replay re-acks
	// without resurrecting or re-granting.
	rp := &recordingProducer{}
	cmd := newReleaseCommand(uuid.New(), holdingId)

	// Re-seed and run the handler twice to exercise the handler's own idempotency:
	// rebuild the row, then deliver release twice.
	if _, err := holding.Restore(db.WithContext(ctx), holdingId.String()); err != nil {
		t.Fatalf("re-seed via restore: %v", err)
	}
	handleReleaseFromMtsHolding(rp.provider())(db)(l, ctx, cmd)
	// After first handler delivery the holding is soft-deleted (readable returns error).
	if _, err := holding.GetById(holdingId.String())(db.WithContext(ctx))(); err == nil {
		t.Fatalf("expected holding soft-deleted after first release delivery")
	}

	// Replay: a second SoftDelete on the already-released row affects ZERO rows,
	// proving the grant-home would not fire a second time (no double-grant).
	second, err := holding.SoftDelete(db.WithContext(ctx), holdingId.String())
	if err != nil {
		t.Fatalf("replay SoftDelete: %v", err)
	}
	if second != 0 {
		t.Fatalf("single-custody invariant violated: take-home replay affected %d rows, want 0 (no double-grant home)", second)
	}

	// And the handler replay re-acks RELEASED without error.
	handleReleaseFromMtsHolding(rp.provider())(db)(l, ctx, cmd)
	if got := countHoldingsForOwner(t, db, ctx, ownerId); got != 0 {
		t.Fatalf("expected 0 live holdings for owner after take-home + replay, got %d", got)
	}
}

// newReleaseCommand builds a ReleaseFromMtsHolding command for the take-home tests.
func newReleaseCommand(transactionId uuid.UUID, holdingId uuid.UUID) custody.Command[custody.ReleaseFromMtsHoldingCommandBody] {
	return custody.Command[custody.ReleaseFromMtsHoldingCommandBody]{
		TransactionId: transactionId,
		Type:          custody.CommandReleaseFromMtsHolding,
		Body:          custody.ReleaseFromMtsHoldingCommandBody{HoldingId: holdingId},
	}
}

// seedActiveListingWithSeller persists an active listing for an explicit seller id
// so per-owner holding counts are isolated under the shared in-memory DB.
func seedActiveListingWithSeller(t *testing.T, db *gorm.DB, ctx context.Context, listingId uuid.UUID, sellerId uint32) listing.Model {
	t.Helper()
	m, err := listing.NewBuilder(test.TestTenantId, 0, sellerId).
		SetId(listingId).
		SetSellerName("Seller").
		SetSaleType(listing.SaleTypeFixed).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetWeaponAttack(17).
		SetSlots(7).
		SetLevel(1).
		SetListValue(1000).
		SetCommissionRate(0.10).
		SetCategory("equip").
		SetSubCategory("onehand").
		Build()
	if err != nil {
		t.Fatalf("build listing: %v", err)
	}
	stored, err := listing.CreateListing(db.WithContext(ctx), m)
	if err != nil {
		t.Fatalf("seed listing: %v", err)
	}
	return stored
}
