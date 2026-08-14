package handler

import (
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// expirationExtenderCashSlotItemType returns the version-scoped
// CashSlotItemType for item classification 550 (item-expiration extenders,
// the Magical Sandglass family).
//
// This MUST remain version-scoped rather than a single constant. Plain 61 is
// also the otherCategory==7 megaphone arm on GMS >= 95, and plain 62 is
// classification 551 on GMS < 95 — a bare literal on either side would route
// another family's sub-body into this arm and mis-decode it.
//
// IDA-verified: gms_v72 get_cashslot_item_type @0x49FB33, gms_v79 @0x47EC3E,
// gms_v83 @0x48645B and gms_v87 @0x473D96 all map case 550 -> 61; gms_v95
// @0x488C70 maps it to 62. gms_v48 and gms_v61 have no arm for the family at
// all (their SendConsumeCashItemUseRequest switches cover types 12-47 and
// 12-52 respectively), so the arm is simply unreachable there.
func expirationExtenderCashSlotItemType(t tenant.Model) CashSlotItemType {
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		return CashSlotItemTypeExpirationExtenderV95
	}
	return CashSlotItemTypeExpirationExtender
}

// extensionTarget is the subset of the target equip's state the eligibility
// gates need, lifted out of asset.Model and the equipment data so the
// decision is a pure function.
type extensionTarget struct {
	Expiration time.Time
	Locked     bool
	CashId     int64
	NotExtend  bool
}

// extensionOutcome is the result of evaluating an extender use. A non-empty
// Reason means the use is rejected: nothing is consumed and nothing is
// mutated.
type extensionOutcome struct {
	Expiration time.Time
	Reason     string
}

// evaluateExpirationExtension applies the client's own gates and formula.
//
// Formula (CDraggableItem::ModifyEquipItem, gms_v83 @0x4F4BB7):
//
//	cap      = now + maxDays*24h        // anchored to NOW, not to the target
//	proposed = target.Expiration + addTime seconds
//	accept iff proposed <= cap
//
// An over-cap use is REJECTED, never clamped-and-consumed: the client shows
// "You cannot extend the effective date beyond %d days" and sends nothing.
// Per the human ruling on this feature, a forged over-cap command is
// rejected server-side (Task 8's atlas-inventory re-derivation) and the
// already-destroyed extender is refunded by the orchestrator's compensator —
// burning a player's extender for a partial grant the client refused is a
// visible loss.
func evaluateExpirationExtension(now time.Time, target extensionTarget, addTime uint32, maxDays uint32) extensionOutcome {
	if target.Expiration.IsZero() {
		return extensionOutcome{Reason: "target is permanent and has no time limit to extend"}
	}
	if target.CashId != 0 {
		return extensionOutcome{Reason: "target is a cash equip"}
	}
	if target.Locked {
		return extensionOutcome{Reason: "target expiration is a sealing-lock window, not a time limit"}
	}
	if target.NotExtend {
		return extensionOutcome{Reason: "target is flagged notExtend"}
	}
	if maxDays == 0 {
		return extensionOutcome{Reason: "extender has no maxDays ceiling"}
	}
	ceiling := now.Add(time.Duration(maxDays) * 24 * time.Hour)
	proposed := target.Expiration.Add(time.Duration(addTime) * time.Second)
	if proposed.After(ceiling) {
		return extensionOutcome{Reason: "extension would push the expiration past the maxDays ceiling"}
	}
	return extensionOutcome{Expiration: proposed}
}
