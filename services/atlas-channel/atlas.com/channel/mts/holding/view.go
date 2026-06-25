package holding

import (
	"time"

	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// ToMtsItem maps one channel-side holding.Model to a clientbound MtsItem
// (ITCITEM) for the GET_USER_PURCHASE_ITEM_DONE ("Transfer Inventory") list. The
// item-slot blob carries the template id and quantity; the MTS trailer carries
// itcSn (= the holding serial, which addresses the take-home arm). A holding has
// no price/bid metadata, so the remaining trailer fields are zeroed.
//
// zeroPosition=true: the ITCITEM's GW_ItemSlotBase blob is bare (the v83 client's
// GW_ItemSlotBase::Decode reads the item type byte first, with NO leading
// inventory-slot byte). A slot-prefixed blob is misread as the item type and
// overruns a later DecodeStr → client crash on MTS entry.
//
// Shared by the entry push (socket/handler) and the post-take-home re-push
// (kafka/consumer/mts) so both produce identical wire bytes.
// mtsHoldingExpiry is the "Sold Until" FILETIME the client displays for a
// taken-home holding (which never expires). A zero FILETIME renders as
// "1-1-01"; this far-future date renders as an effectively-permanent entry.
var mtsHoldingExpiry = time.Date(2079, 1, 1, 0, 0, 0, 0, time.UTC)

func ToMtsItem(m Model) fieldcb.MtsItem {
	item := packetmodel.NewAsset(true, 0, m.TemplateId(), time.Time{}).SetStackableInfo(m.Quantity(), 0, 0)
	dateExpired := packetmodel.MsTimeBytes(mtsHoldingExpiry)
	return fieldpkt.MtsOperationNewItem(
		item,        // GW_ItemSlotBase blob
		m.ItcSn(),   // nITCSN = the holding serial (addresses take-home)
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
