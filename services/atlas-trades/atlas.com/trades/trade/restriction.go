package trade

import (
	"errors"
	"math"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
)

// Named rules so the rejection log says which one fired (design §7). A staging
// refusal is deliberately SILENT to the client — the reference client has no
// put-item-time error for "this item cannot be traded", and the empty dialog
// slot is the feedback — so the rule name is the only diagnostic there is.
var (
	errUntradeableFlag  = errors.New("trade: asset carries an untradeable flag")
	errTradeBlock       = errors.New("trade: item data sets tradeBlock")
	errEquipped         = errors.New("trade: equipped items cannot be staged")
	errUnknownInventory = errors.New("trade: source compartment is not a stageable inventory")
	errItemDataUnknown  = errors.New("trade: item data could not be read")
)

// assetView is the inventory-side input a restriction check needs, decoupled
// from the REST models so the rules are testable without a server.
type assetView struct {
	// Flags is the asset's raw flag bitfield, read through
	// libs/atlas-constants/asset's Flag constants.
	Flags uint16
	// SourceSlot is the inventory position the asset occupies. A NEGATIVE
	// position is the equipped compartment (FR-4.4): atlas-inventory stores
	// worn equipment in the EQUIP compartment at negative slots and filters
	// them out of every bag operation on exactly that test
	// (services/atlas-inventory/atlas.com/inventory/compartment/processor.go:1545-1550,
	// libs/atlas-constants/inventory/slot's Slots table is entirely negative).
	// There is no separate EQUIPPED inventory.Type to compare against.
	SourceSlot slot.Position
	// TemplateId is needed to resolve the KARMA BIT, which is slot-class
	// dependent: 0x10 on an equip, 0x02 on a bundle — and 0x02 on an equip is
	// FlagSpikes. See libs/atlas-constants/asset.KarmaFlagFor.
	TemplateId uint32
}

// itemDataView is the atlas-data side of the same pair: what the WZ item record
// says about tradeability, and whether it could be read at all.
type itemDataView struct {
	TradeBlock bool
	// Unreadable is true when the atlas-data lookup FAILED. A failure is a
	// refusal, not a "tradeable" default (PRD FR-4.2 is explicit about this).
	Unreadable bool
}

// stageableInventoryType decodes the client's raw inventory-type byte into a
// shared inventory.Type, reporting false for anything that is not one of the
// five compartments the service recognises.
//
// The decode is load-bearing twice over. inventory.Type is a SIGNED int8, so a
// byte above 127 arrives negative and would silently address a nonexistent
// compartment if it were merely converted. And FR-4.3 asks for quest items to
// be unstageable: this codebase models exactly five compartments
// (libs/atlas-constants/inventory/constants.go:11-17) and the reference client
// drags from exactly those five, so a QUEST compartment — on any client version
// that grows one — is by construction not among them and is refused here.
func stageableInventoryType(b byte) (inventory.Type, bool) {
	if b > math.MaxInt8 {
		return 0, false
	}
	t := inventory.Type(b)
	for _, known := range inventory.Types {
		if t == known {
			return t, true
		}
	}
	return 0, false
}

// checkRestrictions evaluates FR-4.1..FR-4.4 at stage time. A non-nil error
// means the stage is dropped: no clientbound update, empty client slot, and a
// server-side log naming the item and the failing rule.
//
// source is the RAW inventory-type byte off the wire, not a converted
// inventory.Type — validating it is one of this function's jobs.
func checkRestrictions(a assetView, d itemDataView, source byte) error {
	if _, ok := stageableInventoryType(source); !ok {
		return errUnknownInventory
	}
	if a.SourceSlot < 0 {
		return errEquipped
	}
	// A karma mark (Scissors of Karma, task-223) buys exactly one transfer, and
	// it must defeat BOTH tradeability rules or it defeats nothing useful:
	// untradeable items derive their untradeability mostly from the WZ
	// tradeBlock prop, not from the flag. The mark is CONSUMED by the transfer
	// — it is masked off the settlement snapshot at the moment the receiving
	// asset is built (atlas-saga-orchestrator's trade-settlement expansion), so
	// the item arrives untradeable for its new owner. An UNWOUND (cancelled)
	// trade replays the same snapshot unmasked, so a staged-then-unstaged item
	// keeps its mark.
	//
	// The other three rules are untouched: unknown compartment and equipped slot
	// are checked above, and errItemDataUnknown stays ABOVE the tradeBlock check
	// so an unreadable lookup is never rescued by a mark.
	karmaMarked := false
	if f, ok := asset.KarmaFlagFor(a.TemplateId); ok {
		karmaMarked = asset.HasFlag(a.Flags, f)
	}
	if !karmaMarked && (asset.HasFlag(a.Flags, asset.FlagUntradeable) || asset.HasFlag(a.Flags, asset.FlagMergeUntradeable)) {
		return errUntradeableFlag
	}
	if d.Unreadable {
		return errItemDataUnknown
	}
	if !karmaMarked && d.TradeBlock {
		return errTradeBlock
	}
	return nil
}
