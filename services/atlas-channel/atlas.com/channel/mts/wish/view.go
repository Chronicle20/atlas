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
// (CITC::OnCancelWish, v83 0x59fb07). The wish's price is rendered into the
// nPrice/nMinPrice/nUnitPrice trailer fields so the Cart / Wanted views show the
// real price; the bid fields and nMaxPrice are zeroed (a wish carries no bid /
// price-range metadata).
//
// zeroPosition=true: the ITCITEM's GW_ItemSlotBase blob is bare (the v83 client's
// GW_ItemSlotBase::Decode reads the item type byte first, with NO leading
// inventory-slot byte). A slot-prefixed blob is misread as the item type and
// overruns a later DecodeStr → client crash.
//
// Shared by the synchronous VIEW_WISH arm (socket/handler) and the post-mutation
// Cart/Wanted re-push (kafka/consumer/mts) so both produce identical wire bytes.
func ToMtsItem(m Model) fieldcb.MtsItem {
	return toMtsItem(m, "")
}

// ToMtsItemWithSeller is the cross-character Wanted-tab variant of ToMtsItem: it
// renders the want-ad owner's display name into the sGameID field (the browse's
// seller column) while keeping every other wire field identical to ToMtsItem. A
// want-ad has an owner other than the viewer, so the Wanted browse shows who is
// looking for the item; the Cart / VIEW_WISH views (the viewer's own entries)
// leave sGameID empty via ToMtsItem.
func ToMtsItemWithSeller(m Model, sellerName string) fieldcb.MtsItem {
	return toMtsItem(m, sellerName)
}

// toMtsItem is the shared body: it builds the wish ITCITEM with the given
// sGameID (seller name). All callers keep the same zeroPosition/dateExpired and
// nPrice/nMinPrice/nUnitPrice = the wish price layout; only sGameID varies.
func toMtsItem(m Model, sellerName string) fieldcb.MtsItem {
	item := packetmodel.NewAsset(true, 0, m.ItemId(), time.Time{}).SetStackableInfo(1, 0, 0)
	// A "wanted" want-ad carries a real expiry (created_at + the tenant fixed-sale
	// term); render it as the "Sold Until" date so the client shows a genuine
	// countdown. A "cart" entry has no expiry — keep the far-future 2079 sentinel
	// so it renders as an effectively-permanent entry.
	expiry := mtsWishExpiry
	if m.ExpiresAt() != nil {
		expiry = *m.ExpiresAt()
	}
	dateExpired := packetmodel.MsTimeBytes(expiry)
	return fieldpkt.MtsOperationNewItem(
		item,        // GW_ItemSlotBase blob
		m.Serial(),  // nITCSN = the wish entry's per-(tenant, world) ITC serial
		m.Price(),   // nPrice
		0,           // nContractFee
		"",          // sContractFeeTxId
		"",          // sRollbackUsageID
		dateExpired, // ftITCDateExpired
		"",          // sUserID
		sellerName,  // sGameID = the want-ad owner's name (empty for the viewer's own entries)
		"",          // sComment
		0,           // nBidCount
		0,           // nBidRange
		0,           // nBidPrice
		m.Price(),   // nMinPrice
		0,           // nMaxPrice
		m.Price(),   // nUnitPrice
		fieldcb.MtsProcessStatusNone, // nProcessStatus (want-ads have no history/auction status)
	)
}
