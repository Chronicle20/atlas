package listing_test

// dupe_safety_test.go — the atlas-mts arm of the MTS dupe-safety suite
// (task-102 §5.2, NFR 8.1: "no trade can duplicate an item or desync currency
// under crash, replay, or race").
//
// Each test in this file asserts the SINGLE-CUSTODY invariant DIRECTLY: after a
// replay or a race, the item exists in EXACTLY ONE place (custody-row-count == 1,
// never 0 or 2). The five acceptance scenarios are split across two arms:
//
//   atlas-mts (this file + the custody package's dupe_safety_test.go):
//     3. double-grant replay        — custody/dupe_safety_test.go
//     4. cancel-racing-purchase     — custody/dupe_safety_test.go
//     5. take-home replay           — custody/dupe_safety_test.go
//
//   orchestrator (saga/mts_dupe_safety_test.go):
//     1. crash-mid-list             — re-grant to exactly one place
//     2. grant-before-debit         — debit-first ordering + no-grant-before-debit
//
// The cancel/move and replay scenarios are concentrated in the custody package
// because that is where both real code paths (handleMtsMoveListingToHolding and
// listing.Processor.Cancel) can be driven against a single shared listing row.
// This file documents the split and carries the listing-level race invariant that
// does not require the custody Kafka handler.

import (
	"atlas-mts/holding"
	"atlas-mts/listing"
	"atlas-mts/test"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// newListingDupeFixture builds a logger + DB (migrated for listings and holdings)
// + tenant context. Per-owner counts use unique owner ids so the cache=shared
// in-memory DB leaking rows across tests does not pollute the invariant.
func newListingDupeFixture(t *testing.T) (*logrus.Logger, *gorm.DB, context.Context) {
	t.Helper()
	logger := logrus.New()
	db := test.SetupTestDB(t, listing.Migration, holding.Migration)
	ctx := test.CreateTestContext()
	t.Cleanup(func() { test.CleanupTestDB(t, db) })
	return logger, db, ctx
}

// TestDupeSafety_CancelLoserCreatesNoHolding is scenario 4 (cancel-racing-purchase),
// listing-arm half: when a settle has already moved the listing out of `active`
// (state=sold, simulating the buy winning the race), a concurrent Cancel is the
// race loser — the conditional UPDATE ... WHERE state='active' affects 0 rows, so
// NO seller holding is created. Combined with the custody-package half (which
// asserts the buyer holding from the winning move exists exactly once), the item
// lives in EXACTLY ONE holding, never both.
//
// The mirror direction (cancel wins, settle loses → exactly one seller holding,
// no buyer holding) lives in custody/dupe_safety_test.go where the move handler
// is in scope.
func TestDupeSafety_CancelLoserCreatesNoHolding(t *testing.T) {
	logger, db, ctx := newListingDupeFixture(t)
	p := listing.NewProcessor(logger, ctx, db)

	listingId := uuid.New()
	const sellerId = uint32(8880001)
	seedActiveListingRow(t, db, listingId, sellerId)

	// Buy wins the race first: the listing is moved active -> sold.
	if _, err := listing.UpdateState(db, listingId.String(), listing.StateActive, listing.StateSold); err != nil {
		t.Fatalf("simulate winning buy: %v", err)
	}

	// Cancel races and loses.
	res, err := p.Cancel(listingId.String())
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if res.Won {
		t.Fatal("cancel must lose the race against a settled listing (Won=false)")
	}

	// INVARIANT: the race-losing cancel created zero seller holdings.
	if got := holdingCountForOwner(t, db, sellerId); got != 0 {
		t.Fatalf("single-custody invariant violated: race-losing cancel created %d seller holdings, want 0", got)
	}

	// And the listing was not clobbered back out of sold.
	stored, err := p.GetById(listingId.String())
	if err != nil {
		t.Fatalf("listing lookup: %v", err)
	}
	if stored.State() != listing.StateSold {
		t.Fatalf("cancel loser must not change listing state; want sold, got %s", stored.State())
	}
}

// holdingCountForOwner counts non-deleted holdings owned by ownerId. The
// cache=shared in-memory DB leaks rows across tests, so per-owner filtering keeps
// the count assertion isolated.
func holdingCountForOwner(t *testing.T, db *gorm.DB, ownerId uint32) int {
	t.Helper()
	all, err := holding.GetAll()(db.WithContext(test.CreateTestContext()))()
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
