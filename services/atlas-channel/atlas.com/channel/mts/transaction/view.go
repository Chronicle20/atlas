package transaction

import (
	"time"

	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// nProcessStatus* are the ITCITEM disposition codes the v83 client renders as
// the My Page -> History status column. CITCWnd_List::Draw (v83 0x5BCCD3)
// reads word `nProcessStatus` off the decoded item and, for the contract
// history tab (category 4 / sub 2), feeds it to CITCWnd_List::GetContractHistoryCode
// (0x5BDBBF), which selects the string:
//
//	0 -> "Sold"        (SP_4750)
//	1 -> "Purchased"   (SP_4751)
//	2 -> "Bid Lost"    (SP_4752)
//	3 -> "Cancelled"   (SP_4753)
//
// IDA-verified against the v95 PDB build (CITCWnd_List::GetContractHistoryCode
// 0x5875D0, same mapping). Sending 0 for every row made a bought item render
// as "Sold" (task-102 live finding). atlas-mts records only purchase/sale, so
// we map those two; bid-lost/cancelled are not produced as history rows today.
const (
	nProcessStatusSold      uint16 = 0
	nProcessStatusPurchased uint16 = 1
)

// processStatusForKind maps the transaction kind (atlas-mts KindPurchase/KindSale)
// to the client's contract-history disposition code. An unrecognized kind falls
// back to "Sold" (0) — the prior behavior — rather than an out-of-range code the
// client would render as an empty column.
func processStatusForKind(kind string) uint16 {
	if kind == "purchase" {
		return nProcessStatusPurchased
	}
	return nProcessStatusSold
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
		item,                            // GW_ItemSlotBase blob
		0,                               // nITCSN (history rows are not addressable)
		m.TotalPrice(),                  // nPrice
		0,                               // nContractFee
		"",                              // sContractFeeTxId
		"",                              // sRollbackUsageID
		dateExpired,                     // ftITCDateExpired (the History "Date" column)
		"",                              // sUserID
		"",                              // sGameID
		"",                              // sComment
		0,                               // nBidCount
		0,                               // nBidRange
		0,                               // nBidPrice
		m.TotalPrice(),                  // nMinPrice
		0,                               // nMaxPrice
		m.TotalPrice(),                  // nUnitPrice
		processStatusForKind(m.Kind()), // nProcessStatus (0=Sold, 1=Purchased)
	)
}
