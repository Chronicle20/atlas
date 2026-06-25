package transaction

import (
	"time"

	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// ToMtsItem maps one channel-side transaction.Model to a clientbound MtsItem
// (ITCITEM) for the My Page -> History list. The item-slot blob carries the
// template id and quantity; the price columns (nPrice/nMinPrice/nUnitPrice)
// carry the settled total, and the "Date" column (ftITCDateExpired) carries the
// settle time. A history row has no serial or live bid metadata, so those are
// zeroed.
//
// zeroPosition=true: the ITCITEM's GW_ItemSlotBase blob is bare (the v83
// client's GW_ItemSlotBase::Decode reads the item type byte first, with NO
// leading inventory-slot byte). A slot-prefixed blob is misread as the item type
// and overruns a later DecodeStr → client crash on MTS entry.
func ToMtsItem(m Model) fieldcb.MtsItem {
	item := packetmodel.NewAsset(true, 0, m.ItemId(), time.Time{}).SetStackableInfo(m.Quantity(), 0, 0)
	dateExpired := packetmodel.MsTimeBytes(m.CreatedAt())
	return fieldpkt.MtsOperationNewItem(
		item,           // GW_ItemSlotBase blob
		0,              // nITCSN (history rows are not addressable)
		m.TotalPrice(), // nPrice
		0,              // nContractFee
		"",             // sContractFeeTxId
		"",             // sRollbackUsageID
		dateExpired,    // ftITCDateExpired (the History "Date" column)
		"",             // sUserID
		"",             // sGameID
		"",             // sComment
		0,              // nBidCount
		0,              // nBidRange
		0,              // nBidPrice
		m.TotalPrice(), // nMinPrice
		0,              // nMaxPrice
		m.TotalPrice(), // nUnitPrice
		0,              // nProcessStatus
	)
}
