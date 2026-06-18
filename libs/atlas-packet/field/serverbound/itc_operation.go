package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Serverbound ITC_OPERATION mode-dispatcher (gms_v83 opcode 0xFD/253). The
// client sends ONE opcode (0xFD) with a leading Encode1(mode) byte that selects
// the marketplace operation, followed by that operation's body. This mirrors
// the clientbound CITC::OnNormalItemResult dispatcher (field/clientbound/
// mts_operation.go) but for the request direction.
//
// Each arm is a discrete per-mode body struct (the dispatcher-family rule:
// EACH mode arm needs its OWN byte fixture; enumerating mode bytes is a false
// pass). The struct's constructor takes the dispatcher mode byte as its FIRST
// argument (never hard-coded) — the per-mode handler resolves that byte from
// the tenant "operations" table (the same config-driven contract as the other
// dispatcher families).
//
// CORE-TRADE arms verified here (gms_v83, MapleStory_dump.exe, IDA port 13342):
//
//	CITC::OnRegisterSaleEntry @0x59ec36 — COutPacket(0xFD) @0x59ec63. Handles
//	    BOTH register-fixed-price (mode 2, arg0==0 branch @0x59ed92) and
//	    register-auction (mode 0x12, arg0==1 branch @0x59ecc8) by its arg0
//	    selector.
//	CITC::OnSaleCurrentItem @0x59ee3f — COutPacket(253) @0x59ee5d, mode 3
//	    @0x59ee6b (sell currently-selected item).
//
// The item-slot blob is written by sub_4E33D8 (@0x4e33d8): Encode1(itemType) +
// virtual RawEncode — the GW_ItemSlotBase contract modeled by the shared
// model.Asset codec (the same blob the clientbound MtsItem embeds).
//
// The buy / buy-now / place-bid / cancel-sale / take-home arms of this same
// dispatcher are INLINED into UI dialog handlers in the v83 client (no
// standalone CITC::OnBuy / OnCancelSaleItem / OnMoveITCPurchaseItemLtoS /
// CITCBidAuctionDlg::OnButtonClicked function exists in the IDB or export);
// they are NOT verified here.

const ItcOperationHandle = "ItcOperationHandle"

// ItcOperation is the leading mode-byte dispatcher. A handler decodes this
// first to read the mode byte, reverse-resolves it against the tenant
// "operations" table, then decodes the matching per-mode body struct. It is a
// production handler helper, NOT an audit candidate — the per-mode body codecs
// (ItcOperationRegisterSale / RegisterAuction / SaleCurrentItem) carry the
// verify markers and link to the dispatcher fnames, so this struct deliberately
// has no packet-audit:fname marker (it would otherwise add a permanently-🟡
// shadow sibling to the worst-of dispatcher op row).
type ItcOperation struct {
	mode byte
}

func (m ItcOperation) Mode() byte {
	return m.mode
}

func (m ItcOperation) Operation() string {
	return ItcOperationHandle
}

func (m ItcOperation) String() string {
	return fmt.Sprintf("mode [%d]", m.mode)
}

func (m ItcOperation) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ItcOperation) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ItcOperationRegisterSale — the mode 2 register-fixed-price arm of
// CITC::OnRegisterSaleEntry (@0x59ec36, arg0==0 branch). After the dispatcher
// opcode COutPacket(0xFD) @0x59ec63, the arg0==0 branch (sub_5A9480 path)
// encodes, in order (all cited to the decompile of CITC::OnRegisterSaleEntry):
//
//	Encode1(2u)               @0x59ed92  dispatcher mode byte (fixed-price)
//	sub_4E33D8(a5, &pkt)      @0x59ed9e  item-slot blob (Encode1 type + RawEncode)
//	Encode4(a4)               @0x59eda9  quantity
//	Encode4(v22)              @0x59edb4  commodityId
//	Encode4(a3)               @0x59edbf  price
//	Encode1(a2[0])            @0x59edca  type
//	Encode1(v21[0])           @0x59edd5  flag
//
// The 110-NX floor guard (a3 > 109) gates a StringPool notice before this; it
// does not change the wire shape.
//
// packet-audit:fname CITC::OnRegisterSaleEntry#RegisterSale
type ItcOperationRegisterSale struct {
	mode        byte
	item        model.Asset // sub_4E33D8 GW_ItemSlotBase blob
	quantity    uint32      // Encode4 a4
	commodityId uint32      // Encode4 v22
	price       uint32      // Encode4 a3
	itemType    byte        // Encode1 a2[0]
	flag        byte        // Encode1 v21[0]
}

func NewItcOperationRegisterSale(mode byte, item model.Asset, quantity uint32, commodityId uint32, price uint32, itemType byte, flag byte) ItcOperationRegisterSale {
	return ItcOperationRegisterSale{mode: mode, item: item, quantity: quantity, commodityId: commodityId, price: price, itemType: itemType, flag: flag}
}

func (m ItcOperationRegisterSale) Mode() byte          { return m.mode }
func (m ItcOperationRegisterSale) Item() model.Asset   { return m.item }
func (m ItcOperationRegisterSale) Quantity() uint32    { return m.quantity }
func (m ItcOperationRegisterSale) CommodityId() uint32 { return m.commodityId }
func (m ItcOperationRegisterSale) Price() uint32       { return m.price }
func (m ItcOperationRegisterSale) ItemType() byte      { return m.itemType }
func (m ItcOperationRegisterSale) Flag() byte          { return m.flag }
func (m ItcOperationRegisterSale) Operation() string   { return ItcOperationHandle }
func (m ItcOperationRegisterSale) String() string {
	return fmt.Sprintf("itc register sale mode [%d] qty [%d] commodity [%d] price [%d] type [%d] flag [%d]", m.mode, m.quantity, m.commodityId, m.price, m.itemType, m.flag)
}

func (m ItcOperationRegisterSale) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		itemCopy := m.item
		w.WriteByte(m.mode)                                // Encode1(2u) @0x59ed92 mode byte
		w.WriteByteArray(itemCopy.Encode(l, ctx)(options)) // sub_4E33D8 item-slot blob @0x59ed9e
		w.WriteInt(m.quantity)                             // Encode4 @0x59eda9 quantity
		w.WriteInt(m.commodityId)                          // Encode4 @0x59edb4 commodityId
		w.WriteInt(m.price)                                // Encode4 @0x59edbf price
		w.WriteByte(m.itemType)                            // Encode1 @0x59edca type
		w.WriteByte(m.flag)                                // Encode1 @0x59edd5 flag
		return w.Bytes()
	}
}

func (m *ItcOperationRegisterSale) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.item.Decode(l, ctx)(r, options)
		m.quantity = r.ReadUint32()
		m.commodityId = r.ReadUint32()
		m.price = r.ReadUint32()
		m.itemType = r.ReadByte()
		m.flag = r.ReadByte()
	}
}

// ItcOperationRegisterAuction — the mode 0x12 register-auction arm of
// CITC::OnRegisterSaleEntry (@0x59ec36, arg0==1 branch). After the dispatcher
// opcode COutPacket(0xFD) @0x59ec63, the arg0==1 branch (sub_5AD76B path)
// encodes, in order (all cited to the decompile of CITC::OnRegisterSaleEntry):
//
//	Encode1(0x12u)            @0x59ecc8  dispatcher mode byte (auction)
//	sub_4E33D8(a5, &pkt)      @0x59ecd4  item-slot blob (Encode1 type + RawEncode)
//	Encode4(a4)               @0x59ecdf  quantity
//	Encode4(v22)              @0x59ecea  commodityId
//	Encode4(arg0)             @0x59ecf5  arg0 (==1 here; the auction selector echo)
//	Encode4(v20)              @0x59ed00  buyNowPrice
//	Encode1(a2[0])            @0x59ed0b  type
//	Encode1(v21[0])           @0x59ed16  flag
//	Encode4(v19)              @0x59ed21  durationHrs
//
// The 24..168-hr duration guard gates a StringPool notice before this; it does
// not change the wire shape.
//
// packet-audit:fname CITC::OnRegisterSaleEntry#RegisterAuction
type ItcOperationRegisterAuction struct {
	mode        byte
	item        model.Asset // sub_4E33D8 GW_ItemSlotBase blob
	quantity    uint32      // Encode4 a4
	commodityId uint32      // Encode4 v22
	selector    uint32      // Encode4 arg0 (==1)
	buyNowPrice uint32      // Encode4 v20
	itemType    byte        // Encode1 a2[0]
	flag        byte        // Encode1 v21[0]
	durationHrs uint32      // Encode4 v19
}

func NewItcOperationRegisterAuction(mode byte, item model.Asset, quantity uint32, commodityId uint32, selector uint32, buyNowPrice uint32, itemType byte, flag byte, durationHrs uint32) ItcOperationRegisterAuction {
	return ItcOperationRegisterAuction{mode: mode, item: item, quantity: quantity, commodityId: commodityId, selector: selector, buyNowPrice: buyNowPrice, itemType: itemType, flag: flag, durationHrs: durationHrs}
}

func (m ItcOperationRegisterAuction) Mode() byte          { return m.mode }
func (m ItcOperationRegisterAuction) Item() model.Asset   { return m.item }
func (m ItcOperationRegisterAuction) Quantity() uint32    { return m.quantity }
func (m ItcOperationRegisterAuction) CommodityId() uint32 { return m.commodityId }
func (m ItcOperationRegisterAuction) Selector() uint32    { return m.selector }
func (m ItcOperationRegisterAuction) BuyNowPrice() uint32 { return m.buyNowPrice }
func (m ItcOperationRegisterAuction) ItemType() byte      { return m.itemType }
func (m ItcOperationRegisterAuction) Flag() byte          { return m.flag }
func (m ItcOperationRegisterAuction) DurationHrs() uint32 { return m.durationHrs }
func (m ItcOperationRegisterAuction) Operation() string   { return ItcOperationHandle }
func (m ItcOperationRegisterAuction) String() string {
	return fmt.Sprintf("itc register auction mode [%d] qty [%d] commodity [%d] buyNow [%d] duration [%d]", m.mode, m.quantity, m.commodityId, m.buyNowPrice, m.durationHrs)
}

func (m ItcOperationRegisterAuction) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		itemCopy := m.item
		w.WriteByte(m.mode)                                // Encode1(0x12u) @0x59ecc8 mode byte
		w.WriteByteArray(itemCopy.Encode(l, ctx)(options)) // sub_4E33D8 item-slot blob @0x59ecd4
		w.WriteInt(m.quantity)                             // Encode4 @0x59ecdf quantity
		w.WriteInt(m.commodityId)                          // Encode4 @0x59ecea commodityId
		w.WriteInt(m.selector)                             // Encode4 @0x59ecf5 arg0 (==1)
		w.WriteInt(m.buyNowPrice)                          // Encode4 @0x59ed00 buyNowPrice
		w.WriteByte(m.itemType)                            // Encode1 @0x59ed0b type
		w.WriteByte(m.flag)                                // Encode1 @0x59ed16 flag
		w.WriteInt(m.durationHrs)                          // Encode4 @0x59ed21 durationHrs
		return w.Bytes()
	}
}

func (m *ItcOperationRegisterAuction) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.item.Decode(l, ctx)(r, options)
		m.quantity = r.ReadUint32()
		m.commodityId = r.ReadUint32()
		m.selector = r.ReadUint32()
		m.buyNowPrice = r.ReadUint32()
		m.itemType = r.ReadByte()
		m.flag = r.ReadByte()
		m.durationHrs = r.ReadUint32()
	}
}

// ItcOperationSaleCurrentItem — the mode 3 sell-currently-selected-item arm
// (CITC::OnSaleCurrentItem @0x59ee3f). After the dispatcher opcode
// COutPacket(253) @0x59ee5d, it encodes, in order (cited to the decompile):
//
//	Encode1(3u)               @0x59ee6b  dispatcher mode byte
//	Encode1(a2)               @0x59ee76  type
//	Encode4(a3)               @0x59ee81  slotPos
//	sub_4E33D8(a4, &pkt)      @0x59ee8d  item-slot blob (Encode1 type + RawEncode)
//	Encode4(a5)               @0x59ee98  commodityId
//
// packet-audit:fname CITC::OnSaleCurrentItem
type ItcOperationSaleCurrentItem struct {
	mode        byte
	itemType    byte        // Encode1 a2
	slotPos     uint32      // Encode4 a3
	item        model.Asset // sub_4E33D8 GW_ItemSlotBase blob
	commodityId uint32      // Encode4 a5
}

func NewItcOperationSaleCurrentItem(mode byte, itemType byte, slotPos uint32, item model.Asset, commodityId uint32) ItcOperationSaleCurrentItem {
	return ItcOperationSaleCurrentItem{mode: mode, itemType: itemType, slotPos: slotPos, item: item, commodityId: commodityId}
}

func (m ItcOperationSaleCurrentItem) Mode() byte          { return m.mode }
func (m ItcOperationSaleCurrentItem) ItemType() byte      { return m.itemType }
func (m ItcOperationSaleCurrentItem) SlotPos() uint32     { return m.slotPos }
func (m ItcOperationSaleCurrentItem) Item() model.Asset   { return m.item }
func (m ItcOperationSaleCurrentItem) CommodityId() uint32 { return m.commodityId }
func (m ItcOperationSaleCurrentItem) Operation() string   { return ItcOperationHandle }
func (m ItcOperationSaleCurrentItem) String() string {
	return fmt.Sprintf("itc sale current item mode [%d] type [%d] slot [%d] commodity [%d]", m.mode, m.itemType, m.slotPos, m.commodityId)
}

func (m ItcOperationSaleCurrentItem) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		itemCopy := m.item
		w.WriteByte(m.mode)                                // Encode1(3u) @0x59ee6b mode byte
		w.WriteByte(m.itemType)                            // Encode1 @0x59ee76 type
		w.WriteInt(m.slotPos)                              // Encode4 @0x59ee81 slotPos
		w.WriteByteArray(itemCopy.Encode(l, ctx)(options)) // sub_4E33D8 item-slot blob @0x59ee8d
		w.WriteInt(m.commodityId)                          // Encode4 @0x59ee98 commodityId
		return w.Bytes()
	}
}

func (m *ItcOperationSaleCurrentItem) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itemType = r.ReadByte()
		m.slotPos = r.ReadUint32()
		m.item.Decode(l, ctx)(r, options)
		m.commodityId = r.ReadUint32()
	}
}
