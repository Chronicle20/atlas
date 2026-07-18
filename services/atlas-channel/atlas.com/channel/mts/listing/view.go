package listing

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// mtsFixedExpiry is the "Sold Until" FILETIME the client displays for a listing
// with no server-side end time (fixed-price sales do not expire in atlas-mts).
// A zero FILETIME renders as "1-1-01"; this far-future date renders as an
// effectively-permanent listing instead.
var mtsFixedExpiry = time.Date(2079, 1, 1, 0, 0, 0, 0, time.UTC)

// mtsItemAsset builds the ITCITEM's GW_ItemSlotBase blob from a listing. For an
// equip it carries the full equipment stat block + upgrade slots (SetEquipmentStats
// / SetEquipmentMeta) so the client renders the item's real, scrolled stats — e.g.
// a +2 weapon-attack cape shows +2, not the base template. For a non-equip it is a
// bundle (stackable) blob. Both paths carry the item-tag owner name (sOwner) so a
// tagged item surfaces its owner in every MTS view. The equip stats + owner are
// stored on the listing by atlas-mts (captured from the seller's asset at list
// time); this is purely the presentation of already-persisted data.
//
// zeroPosition=true: the blob is bare (the v83 client's GW_ItemSlotBase::Decode
// reads the item type byte first, with NO leading inventory-slot byte). A
// slot-prefixed blob is misread as the item type and overruns a later DecodeStr →
// client crash on browse. The equip type byte (0x01) and stackable type byte (0x02)
// are both valid bare leads; only a slot prefix is the crash.
func mtsItemAsset(m Model) packetmodel.Asset {
	asset := packetmodel.NewAsset(true, 0, m.TemplateId(), time.Time{})
	if it, ok := inventory.TypeFromItemId(item.Id(m.TemplateId())); ok && it == inventory.TypeValueEquip {
		asset = asset.
			SetEquipmentStats(m.Strength(), m.Dexterity(), m.Intelligence(), m.Luck(), m.HP(), m.MP(), m.WeaponAttack(), m.MagicAttack(), m.WeaponDefense(), m.MagicDefense(), m.Accuracy(), m.Avoidability(), m.Hands(), m.Speed(), m.Jump()).
			SetEquipmentMeta(m.Slots(), 0, m.Level(), m.ItemExp(), 0, m.Flags())
	} else {
		asset = asset.SetStackableInfo(m.Quantity(), 0, 0)
	}
	return asset.SetOwner(m.Owner())
}

// ToMtsItem maps one channel-side listing.Model to a clientbound MtsItem
// (ITCITEM) for the browse / user-sale page. The item-slot blob (see mtsItemAsset)
// carries the full equipment stats for equips and the owner tag for any item; the
// MTS trailer carries itcSn (= the listing's serial), price, and the auction bid
// metadata. The contract-fee / rollback / user-id strings are empty (the channel
// surfaces no such state). The date-expired FILETIME is the auction end (so the
// bid dialog's countdown is correct) or a far-future sentinel for non-expiring
// fixed listings.
//
// Shared by the browse arm (socket/handler) and the post-event re-push of the
// seller's "Not Yet Sold" list (kafka/consumer/mts) so both produce identical
// wire bytes.
func ToMtsItem(m Model) fieldcb.MtsItem {
	return toMtsItem(m, mtsItemAsset(m))
}

// ToMtsItemDetailed is retained for the want-ad offers view (VIEW_WISH). Since the
// browse view now renders full equipment stats too (mtsItemAsset), it is identical
// to ToMtsItem; the distinct name documents the offer-comparison call site.
func ToMtsItemDetailed(m Model) fieldcb.MtsItem {
	return ToMtsItem(m)
}

// toMtsItem builds the ITCITEM from a listing and a pre-built item-slot asset. The
// asset is the only thing that varies between the browse view (bare stackable blob)
// and the offer-detail view (full equip stats); the MTS trailer is identical.
func toMtsItem(m Model, asset packetmodel.Asset) fieldcb.MtsItem {
	expiry := mtsFixedExpiry
	if m.EndsAt() != nil {
		expiry = *m.EndsAt()
	}
	dateExpired := packetmodel.MsTimeBytes(expiry)

	// The My Page -> Auction tab draws its Category column via
	// CITCWnd_List::GetAuctionHistoryCode(nProcessStatus): "Exhibit" (an auction I
	// listed) vs "Bid" (an auction I bid on), else empty. A seller's own active
	// auction is an Exhibit; without this the column renders blank (task-102 live
	// finding). Fixed sales don't use this column. The wire code is config-resolved
	// from the tenant processStatusCodes table (DOM-25), not a Go literal.
	processStatusKey := fieldcb.MtsProcessStatusNone
	if m.SaleType() == "auction" {
		processStatusKey = fieldcb.MtsProcessStatusAuctionExhibit
	}

	// The client draws the price column as nPrice+nContractFee (fixed) or
	// nBidPrice+nContractFee (auction) — so nContractFee is the buyer-visible fee on
	// top of the base. m.ContractFee() carries markedUp(base)-base from atlas-mts.
	return fieldpkt.MtsOperationNewItem(
		asset,            // GW_ItemSlotBase blob
		m.ItcSn(),        // nITCSN = the listing serial (addresses buy/cancel/bid)
		m.ListValue(),    // nPrice
		m.ContractFee(),  // nContractFee (buyer-visible fee; client adds it to the price)
		"",               // sContractFeeTxId
		"",               // sRollbackUsageID
		dateExpired,      // ftITCDateExpired
		"",               // sUserID
		m.SellerName(),   // sGameID (seller display name)
		"",               // sComment
		m.BidCount(),     // nBidCount (total bids placed on this listing)
		m.MinIncrement(), // nBidRange
		m.CurrentBid(),   // nBidPrice
		m.ListValue(),    // nMinPrice
		m.BuyNowPrice(),  // nMaxPrice
		m.ListValue(),    // nUnitPrice
		processStatusKey, // nProcessStatus (auction => Exhibit; config-resolved)
	)
}
