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

// ToMtsItem maps one channel-side listing.Model to a clientbound MtsItem
// (ITCITEM) for the browse / user-sale page. The item-slot blob carries the
// template id and quantity; the MTS trailer carries itcSn (= the listing's
// serial), price, and the auction bid metadata. The contract-fee / rollback /
// user-id strings are empty (the channel surfaces no such state). The
// date-expired FILETIME is the auction end (so the bid dialog's countdown is
// correct) or a far-future sentinel for non-expiring fixed listings.
//
// zeroPosition=true: the ITCITEM's GW_ItemSlotBase blob is bare (the v83
// client's GW_ItemSlotBase::Decode reads the item type byte first, with NO
// leading inventory-slot byte). A slot-prefixed blob is misread as the item
// type and overruns a later DecodeStr → client crash on browse.
//
// Shared by the browse arm (socket/handler) and the post-event re-push of the
// seller's "Not Yet Sold" list (kafka/consumer/mts) so both produce identical
// wire bytes.
func ToMtsItem(m Model) fieldcb.MtsItem {
	asset := packetmodel.NewAsset(true, 0, m.TemplateId(), time.Time{}).SetStackableInfo(m.Quantity(), 0, 0)
	return toMtsItem(m, asset)
}

// ToMtsItemDetailed is ToMtsItem except the ITCITEM's item-slot blob carries full
// equipment stats when the listed item is an equip, so a want-ad poster viewing the
// offers on their ad (VIEW_WISH) can compare each offered equip's real stats. For
// non-equips it is byte-identical to ToMtsItem (the same stackable blob). Only the
// item asset differs — every trailer field (itcSn, price, contract fee,
// dateExpired, seller name, processStatus) is the same.
func ToMtsItemDetailed(m Model) fieldcb.MtsItem {
	var asset packetmodel.Asset
	if it, ok := inventory.TypeFromItemId(item.Id(m.TemplateId())); ok && it == inventory.TypeValueEquip {
		asset = packetmodel.NewAsset(true, 0, m.TemplateId(), time.Time{}).
			SetEquipmentStats(m.Strength(), m.Dexterity(), m.Intelligence(), m.Luck(), m.HP(), m.MP(), m.WeaponAttack(), m.MagicAttack(), m.WeaponDefense(), m.MagicDefense(), m.Accuracy(), m.Avoidability(), m.Hands(), m.Speed(), m.Jump()).
			SetEquipmentMeta(m.Slots(), 0, m.Level(), m.ItemExp(), 0, m.Flags())
	} else {
		asset = packetmodel.NewAsset(true, 0, m.TemplateId(), time.Time{}).SetStackableInfo(m.Quantity(), 0, 0)
	}
	return toMtsItem(m, asset)
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
