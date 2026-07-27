package transaction

import (
	"time"

	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// processStatusKeyForKind maps the transaction kind (atlas-mts Kind*) to the
// SEMANTIC My Page -> History disposition key. The v83 client renders the
// nProcessStatus column via CITCWnd_List::GetContractHistoryCode (v83 0x5BDBBF /
// v95 0x5875D0, same mapping): Sold / Purchased / Bid Lost / Cancelled. The wire
// codes (0..3) are config-resolved from the tenant processStatusCodes table at
// encode time (DOM-25) — see fieldcb.MtsProcessStatus*. Sending "Sold" for every
// row made a bought item render as "Sold" (task-102 live finding); all four
// dispositions are now mapped. An unrecognized kind falls back to Sold.
func processStatusKeyForKind(kind string) string {
	switch kind {
	case "purchase":
		return fieldcb.MtsProcessStatusHistoryPurchased
	case "bid_lost":
		return fieldcb.MtsProcessStatusHistoryBidLost
	case "cancelled":
		return fieldcb.MtsProcessStatusHistoryCancelled
	default: // "sale" and any unknown kind
		return fieldcb.MtsProcessStatusHistorySold
	}
}

// ToMtsItem maps one channel-side transaction.Model to a clientbound MtsItem
// (ITCITEM) for the My Page -> History list. The item-slot blob carries the
// template id and quantity; the price columns (nPrice/nMinPrice/nUnitPrice)
// carry the settled total, and the "Date" column (ftITCDateExpired) carries the
// settle time. A history row has no serial or live bid metadata, so those are
// zeroed. nProcessStatus carries the buy/sell disposition (see the const block).
//
// zeroPosition=true: the ITCITEM's GW_ItemSlotBase blob is bare (the v83
// client's GW_ItemSlotBase::Decode reads the item type byte first, with NO
// leading inventory-slot byte). A slot-prefixed blob is misread as the item type
// and overruns a later DecodeStr → client crash on MTS entry.
func ToMtsItem(m Model) fieldcb.MtsItem {
	item := packetmodel.NewAsset(true, 0, m.ItemId(), time.Time{}).SetStackableInfo(m.Quantity(), 0, 0)
	dateExpired := packetmodel.MsTimeBytes(m.CreatedAt())
	return fieldpkt.MtsOperationNewItem(
		item,                              // GW_ItemSlotBase blob
		0,                                 // nITCSN (history rows are not addressable)
		m.TotalPrice(),                    // nPrice
		0,                                 // nContractFee
		"",                                // sContractFeeTxId
		"",                                // sRollbackUsageID
		dateExpired,                       // ftITCDateExpired (the History "Date" column)
		"",                                // sUserID
		"",                                // sGameID
		"",                                // sComment
		0,                                 // nBidCount
		0,                                 // nBidRange
		0,                                 // nBidPrice
		m.TotalPrice(),                    // nMinPrice
		0,                                 // nMaxPrice
		m.TotalPrice(),                    // nUnitPrice
		processStatusKeyForKind(m.Kind()), // nProcessStatus (config-resolved disposition)
	)
}
