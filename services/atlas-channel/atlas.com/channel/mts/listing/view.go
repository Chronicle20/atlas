package listing

import (
	"time"

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
	item := packetmodel.NewAsset(true, 0, m.TemplateId(), time.Time{}).SetStackableInfo(m.Quantity(), 0, 0)
	expiry := mtsFixedExpiry
	if m.EndsAt() != nil {
		expiry = *m.EndsAt()
	}
	dateExpired := packetmodel.MsTimeBytes(expiry)
	return fieldpkt.MtsOperationNewItem(
		item,             // GW_ItemSlotBase blob
		m.ItcSn(),        // nITCSN = the listing serial (addresses buy/cancel/bid)
		m.ListValue(),    // nPrice
		0,                // nContractFee
		"",               // sContractFeeTxId
		"",               // sRollbackUsageID
		dateExpired,      // ftITCDateExpired
		"",               // sUserID
		m.SellerName(),   // sGameID (seller display name)
		"",               // sComment
		0,                // nBidCount
		m.MinIncrement(), // nBidRange
		m.CurrentBid(),   // nBidPrice
		m.ListValue(),    // nMinPrice
		m.BuyNowPrice(),  // nMaxPrice
		m.ListValue(),    // nUnitPrice
		0,                // nProcessStatus
	)
}
