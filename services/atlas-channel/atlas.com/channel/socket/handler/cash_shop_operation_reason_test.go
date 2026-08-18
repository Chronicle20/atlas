package handler

import (
	"atlas-channel/pendingchange"
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestWorldTransferRejectionReasonPassesCheckUnavailableThrough pins the
// atlas-character -> atlas-channel side of the cross-service seam described
// in bug-world-transfer-eligibility-reasons.md §2b: atlas-character's
// evaluateTransferEligibility now reports a remote dependency error as
// "check_unavailable" rather than the gate's affirmative reason.
// worldTransferRejectionReason must forward that verbatim to
// CashShopTransferWorldFailedBody -- not collapse it into "unknown_error",
// which would erase the distinction the fix exists to create.
func TestWorldTransferRejectionReasonPassesCheckUnavailableThrough(t *testing.T) {
	err := &pendingchange.RejectedError{Status: 422, Reason: "check_unavailable"}
	got := worldTransferRejectionReason(err)
	if got != "check_unavailable" {
		t.Fatalf("worldTransferRejectionReason = %q, want check_unavailable", got)
	}
}

// TestCashShopTransferWorldFailedBodyResolvesCheckUnavailableAsConfigured
// exercises the BUY-time wire path end to end: worldTransferRejectionReason's
// output feeds CashShopTransferWorldFailedBody, which resolves the reason
// through the tenant's "errors" table exactly like every other reason (task-
// 227 fix-round step 5 seeded check_unavailable -> PLEASE_TRY_AGAIN into every
// applicable template). The NEW contract this pins: the raw "check_unavailable"
// string -- not a folded-down "unknown_error" -- is the key that reaches
// ResolveCode.
func TestCashShopTransferWorldFailedBodyResolvesCheckUnavailableAsConfigured(t *testing.T) {
	err := &pendingchange.RejectedError{Status: 422, Reason: "check_unavailable"}
	reason := worldTransferRejectionReason(err)

	options := map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CashShopOperationTransferWorldFailed: float64(49),
		},
		"errors": map[string]interface{}{
			"check_unavailable": float64(231),
			"unknown_error":     float64(99),
		},
	}

	body := cashcb.CashShopTransferWorldFailedBody(reason)(logrus.New(), context.Background())(options)
	if len(body) != 2 {
		t.Fatalf("body = %#v, want 2 bytes (mode, errorCode)", body)
	}
	if body[0] != 49 {
		t.Fatalf("mode byte = %d, want 49", body[0])
	}
	if body[1] != 231 {
		t.Fatalf("errorCode byte = %d, want 231 (check_unavailable's configured code) -- got %d, which is unknown_error's code: the reason was folded down instead of passed through", body[1], body[1])
	}
}

// TestCheckTransferWorldPossibleReasonKeyRoutesCheckUnavailableToUnknownError
// asserts the CHECK-time side of the same seam (the reason -> arm mapper in
// libs/atlas-packet, called checkTransferWorldPossibleReasonKey internally):
// check_unavailable is a real member of the closed taxonomy (design §6) but
// is not one of the arms with independently confirmed distinct client text,
// so it must resolve to the client's UNKNOWN_ERROR default arm exactly like
// an unrecognised reason would, never silently to some other arm. The CHECK
// handler itself does not call this path yet (step 4, deferred); this pins
// the mapper's own contract so the wiring in step 4 has a known-correct
// target.
func TestCheckTransferWorldPossibleReasonKeyRoutesCheckUnavailableToUnknownError(t *testing.T) {
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CheckTransferWorldPossibleUnknownError: float64(9),
			cashcb.CheckTransferWorldPossibleInFamily:     float64(8),
		},
	}

	ctx := tenant.WithContext(context.Background(), mustTenant(t, "GMS", 83, 1))
	body := cashcb.CheckTransferWorldPossibleResultRejectedBody(1, "check_unavailable", nil)(logrus.New(), ctx)(options)
	if len(body) < 5 {
		t.Fatalf("body = %#v, too short to carry the result byte", body)
	}
	// characterId(4) + result(1) is the fixed prefix on every GMS version.
	if body[4] != 9 {
		t.Fatalf("result byte = %d, want 9 (UNKNOWN_ERROR's configured code)", body[4])
	}
}

// TestCheckTransferWorldPossibleReasonKeyRoutesInFamilyToItsOwnArm is this
// test file's step-4 seam pin (design's OQ-7 split,
// docs/tasks/task-227-cash-name-change-world-transfer/bug-world-transfer-eligibility-reasons.md,
// "The better fix for 2c"), complementing
// TestCheckTransferWorldPossibleReasonKeyRoutesCheckUnavailableToUnknownError
// immediately above. The NEW contract step 4 exists to deliver: unlike
// check_unavailable (and every other reason on this op besides in_family),
// "in_family" -- reported by atlas-character's destination-independent gate
// via CashShopCheckTransferWorldPossibleHandleFunc's
// checkPossibleTransferEligibilityIndependentFunc seam -- resolves to its own
// confirmed arm (IN_FAMILY, StringPool 5017) rather than folding to the
// generic UNKNOWN_ERROR. TestTransferWorldPossibleRejectsOnIndependentGate
// (cash_shop_check_transfer_world_possible_test.go) exercises the full
// handler-level wiring of this same seam; this test pins the mapper's own
// contract for the specific string the handler passes through unmodified.
func TestCheckTransferWorldPossibleReasonKeyRoutesInFamilyToItsOwnArm(t *testing.T) {
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CheckTransferWorldPossibleUnknownError: float64(9),
			cashcb.CheckTransferWorldPossibleInFamily:     float64(8),
		},
	}

	ctx := tenant.WithContext(context.Background(), mustTenant(t, "GMS", 83, 1))
	body := cashcb.CheckTransferWorldPossibleResultRejectedBody(1, "in_family", nil)(logrus.New(), ctx)(options)
	if len(body) < 5 {
		t.Fatalf("body = %#v, too short to carry the result byte", body)
	}
	if body[4] != 8 {
		t.Fatalf("result byte = %d, want 8 (IN_FAMILY's configured code) -- in_family must not fold to UNKNOWN_ERROR at CHECK time", body[4])
	}
}
