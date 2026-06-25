package wish

import (
	"time"

	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// mtsWishExpiry is the "Sold Until" FILETIME the client displays for a wish
// (cart/wanted) entry, which never expires. A zero FILETIME renders as "1-1-01";
// this far-future date renders as an effectively-permanent entry.
var mtsWishExpiry = time.Date(2079, 1, 1, 0, 0, 0, 0, time.UTC)

// ToMtsItem maps one channel-side wish.Model to a minimal clientbound MtsItem
// (ITCITEM) for the Cart / Wanted views (LoadWishSaleListDone and the
// post-mutation GetItcListDone re-push). The item-slot blob carries the wish's
// item template (quantity 1); the MTS trailer carries the wish's per-(tenant,
// world) ITC serial as nITCSN so the client can echo it back on CANCEL_WISH
// (CITC::OnCancelWish, v83 0x59fb07). A wish has no price/bid metadata, so the
// remaining trailer fields are zeroed.
//
// zeroPosition=true: the ITCITEM's GW_ItemSlotBase blob is bare (the v83 client's
// GW_ItemSlotBase::Decode reads the item type byte first, with NO leading
// inventory-slot byte). A slot-prefixed blob is misread as the item type and
// overruns a later DecodeStr → client crash.
//
// Shared by the synchronous VIEW_WISH arm (socket/handler) and the post-mutation
// Cart/Wanted re-push (kafka/consumer/mts) so both produce identical wire bytes.
func ToMtsItem(m Model) fieldcb.MtsItem {
	item := packetmodel.NewAsset(true, 0, m.ItemId(), time.Time{}).SetStackableInfo(1, 0, 0)
	dateExpired := packetmodel.MsTimeBytes(mtsWishExpiry)
	return fieldpkt.MtsOperationNewItem(
		item,        // GW_ItemSlotBase blob
		m.Serial(),  // nITCSN = the wish entry's per-(tenant, world) ITC serial
		0,           // nPrice
		0,           // nContractFee
		"",          // sContractFeeTxId
		"",          // sRollbackUsageID
		dateExpired, // ftITCDateExpired
		"",          // sUserID
		"",          // sGameID
		"",          // sComment
		0,           // nBidCount
		0,           // nBidRange
		0,           // nBidPrice
		0,           // nMinPrice
		0,           // nMaxPrice
		0,           // nUnitPrice
		0,           // nProcessStatus
	)
}
