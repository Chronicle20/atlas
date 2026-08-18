package pending_change

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TestPurchasePathMintsAndConsumesNoAsset pins the other half of the contract
// documented on Entity.AssetId ("AssetId is null on the cash-shop purchase
// path, which carries an entitlement reference correlated by TransactionId
// instead of an asset").
//
// Every other test in this package creates with a non-nil assetId, so until now
// nothing pinned what the purchase path — the only path atlas-channel actually
// produces — must do: emit no destroy_asset on create, and, crucially, no
// award_asset on cancel. The live regression on atlas-pr-1370 was exactly a
// record that took the item path's emits while being a purchase, leaving the
// player holding two coupons after cancelling one name change. See
// docs/tasks/task-227-cash-name-change-world-transfer/bug-purchase-path-sets-assetid.md.
func TestPurchasePathMintsAndConsumesNoAsset(t *testing.T) {
	for _, tc := range []struct {
		name       string
		changeType string
		requested  string
		destWorld  world.Id
	}{
		{name: "name change", changeType: TypeNameChange, requested: "Tango", destWorld: world.Id(0)},
		{name: "world transfer", changeType: TypeWorldTransfer, requested: "", destWorld: world.Id(2)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newProcessorTestDB(t)
			characterId := seedCharacter(t, db, "Romeo", world.Id(0))
			p := NewProcessor(testLogger(t), testContext(t), db).withTransferEligibilityGates(passingGateDeps())

			// assetId nil is what makes this the purchase path.
			m, err := p.CreateAndEmit(uuid.New(), characterId, tc.changeType, tc.requested, tc.destWorld, nil)
			if err != nil {
				t.Fatalf("CreateAndEmit: %v", err)
			}
			if m.HasAsset() {
				t.Fatal("a purchase-path record must not carry an asset")
			}
			if got := countOutboxMessagesMatching(t, db, "PENDING_CHANGE_CREATED"); got != 1 {
				t.Fatalf("expected 1 created event, got %d", got)
			}
			if got := countOutboxMessagesMatching(t, db, "destroy_asset"); got != 0 {
				t.Fatalf("purchase path has no coupon to consume, got %d destroy_asset commands", got)
			}

			// The discriminating half: cancelling must not mint a coupon.
			got, moved, err := p.CancelForCharacterAndType(characterId, tc.changeType)
			if err != nil {
				t.Fatalf("CancelForCharacterAndType: %v", err)
			}
			if !moved || got.Status() != StatusCancelled {
				t.Fatalf("expected a cancelled record, got moved=%v status=%s", moved, got.Status())
			}
			if n := countOutboxMessagesMatching(t, db, "award_asset"); n != 0 {
				t.Fatalf("cancelling a purchase-path record must refund no asset, got %d award_asset commands", n)
			}
		})
	}
}
