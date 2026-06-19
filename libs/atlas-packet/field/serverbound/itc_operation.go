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
// BUY / BUY-NOW / CANCEL-SALE / TAKE-HOME / PLACE-BID arms verified here
// (gms_v95, GMS_v95.0_U_DEVM.exe, IDA port 13340 — the symbol-rich PDB build
// exposes these as named CITC::On* functions; the v83 client inlines them into
// UI dialog handlers with no standalone fname, so they were BLOCKED on v83).
// All five send the dispatcher opcode COutPacket(308/0x134) then a leading
// Encode1(mode) byte, derived per-function below:
//
//	CITC::OnBuy @0x573270 — COutPacket(308) @0x5732a5, Encode1(0x10) @0x5732b8,
//	    Encode4(nITCSN) @0x5732cc. Mode 0x10 (buy fixed-price).
//	CITC::OnBuyAuctionImm @0x573310 — COutPacket(308) @0x573345,
//	    Encode1(0x14) @0x573358, Encode4(nITCSN) @0x57336c. Mode 0x14 (buy-now).
//	CITC::OnCancelSaleItem @0x5737a0 — COutPacket(308) @0x57381a,
//	    Encode1(7) @0x57382d, Encode4(nITCSN) @0x57383d. Mode 0x07 (cancel sale).
//	CITC::OnMoveITCPurchaseItemLtoS @0x573880 — COutPacket(308) @0x5738b5,
//	    Encode1(8) @0x5738c8, Encode4(nITCSN) @0x5738dc. Mode 0x08 (take-home).
//	    (The nTI/nPos args are NOT written to the wire — only nITCSN.)
//	CITCBidAuctionDlg::OnButtonClicked @0x58eb50 — the nId==1 (confirm-bid)
//	    branch inlines the send: COutPacket(308) @0x58eda1, Encode1(0x13)
//	    @0x58edb4, Encode4(nITCSN) @0x58edc7, Encode4(m_nMyBidPrice) @0x58edd7,
//	    Encode4(m_nMyBidRange) @0x58ede7. Mode 0x13 (place-bid).
//
// Each of these five carries the same body shape across all versions (per-
// version opcode + mode bytes); v95 is the symbol-rich reference for
// propagation to v83/v84/v87/jms by matching the inlined send sites by shape.

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
// packet-audit:fname CITC::OnRegisterSaleEntry
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

// ItcOperationBuy — the buy-fixed-price arm (CITC::OnBuy @0x573270, gms_v95).
// After the dispatcher opcode COutPacket(308) @0x5732a5 it encodes, in order:
//
//	Encode1(0x10u)            @0x5732b8  dispatcher mode byte (buy)
//	Encode4(ii->p->nITCSN)   @0x5732cc  the ITC serial number of the listing
//
// No item-slot blob — a fixed-price purchase references the listing solely by
// its serial. The m_bITCRequestSent latch (@0x573296) guards a double-send; it
// is not written to the wire.
//
// packet-audit:fname CITC::OnBuy
type ItcOperationBuy struct {
	mode  byte
	itcSn uint32 // Encode4 ii->p->nITCSN
}

func NewItcOperationBuy(mode byte, itcSn uint32) ItcOperationBuy {
	return ItcOperationBuy{mode: mode, itcSn: itcSn}
}

func (m ItcOperationBuy) Mode() byte        { return m.mode }
func (m ItcOperationBuy) ItcSn() uint32     { return m.itcSn }
func (m ItcOperationBuy) Operation() string { return ItcOperationHandle }
func (m ItcOperationBuy) String() string {
	return fmt.Sprintf("itc buy mode [%d] itcSn [%d]", m.mode, m.itcSn)
}

func (m ItcOperationBuy) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // Encode1(0x10u) @0x5732b8 mode byte
		w.WriteInt(m.itcSn) // Encode4 @0x5732cc nITCSN
		return w.Bytes()
	}
}

func (m *ItcOperationBuy) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itcSn = r.ReadUint32()
	}
}

// ItcOperationBuyAuctionImm — the buy-now-on-auction arm
// (CITC::OnBuyAuctionImm @0x573310, gms_v95). After COutPacket(308) @0x573345
// it encodes, in order:
//
//	Encode1(0x14u)            @0x573358  dispatcher mode byte (buy-now)
//	Encode4(ii->p->nITCSN)   @0x57336c  the ITC serial number of the listing
//
// Identical body shape to OnBuy (serial-only); the mode byte distinguishes the
// immediate-buy-out of an auction from a plain fixed-price buy.
//
// packet-audit:fname CITC::OnBuyAuctionImm
type ItcOperationBuyAuctionImm struct {
	mode  byte
	itcSn uint32 // Encode4 ii->p->nITCSN
}

func NewItcOperationBuyAuctionImm(mode byte, itcSn uint32) ItcOperationBuyAuctionImm {
	return ItcOperationBuyAuctionImm{mode: mode, itcSn: itcSn}
}

func (m ItcOperationBuyAuctionImm) Mode() byte        { return m.mode }
func (m ItcOperationBuyAuctionImm) ItcSn() uint32     { return m.itcSn }
func (m ItcOperationBuyAuctionImm) Operation() string { return ItcOperationHandle }
func (m ItcOperationBuyAuctionImm) String() string {
	return fmt.Sprintf("itc buy-now mode [%d] itcSn [%d]", m.mode, m.itcSn)
}

func (m ItcOperationBuyAuctionImm) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // Encode1(0x14u) @0x573358 mode byte
		w.WriteInt(m.itcSn) // Encode4 @0x57336c nITCSN
		return w.Bytes()
	}
}

func (m *ItcOperationBuyAuctionImm) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itcSn = r.ReadUint32()
	}
}

// ItcOperationCancelSale — the cancel-sale arm (CITC::OnCancelSaleItem
// @0x5737a0, gms_v95). A YesNo confirm dialog (StringPool 0x12BA, @0x57380f)
// gates the send; the wire shape after COutPacket(308) @0x57381a is:
//
//	Encode1(7u)              @0x57382d  dispatcher mode byte (cancel sale)
//	Encode4(ii->p->nITCSN)   @0x57383d  the ITC serial number of the listing
//
// The cancel is suppressed when the listing already has bids
// (!ii->p->nBidCount guard @0x5737d8); that guard does not change the wire.
//
// packet-audit:fname CITC::OnCancelSaleItem
type ItcOperationCancelSale struct {
	mode  byte
	itcSn uint32 // Encode4 ii->p->nITCSN
}

func NewItcOperationCancelSale(mode byte, itcSn uint32) ItcOperationCancelSale {
	return ItcOperationCancelSale{mode: mode, itcSn: itcSn}
}

func (m ItcOperationCancelSale) Mode() byte        { return m.mode }
func (m ItcOperationCancelSale) ItcSn() uint32     { return m.itcSn }
func (m ItcOperationCancelSale) Operation() string { return ItcOperationHandle }
func (m ItcOperationCancelSale) String() string {
	return fmt.Sprintf("itc cancel sale mode [%d] itcSn [%d]", m.mode, m.itcSn)
}

func (m ItcOperationCancelSale) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // Encode1(7u) @0x57382d mode byte
		w.WriteInt(m.itcSn) // Encode4 @0x57383d nITCSN
		return w.Bytes()
	}
}

func (m *ItcOperationCancelSale) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itcSn = r.ReadUint32()
	}
}

// ItcOperationMoveLtoS — the take-home (move purchased item locker->slot) arm
// (CITC::OnMoveITCPurchaseItemLtoS @0x573880, gms_v95). After COutPacket(308)
// @0x5738b5 it encodes, in order:
//
//	Encode1(8u)              @0x5738c8  dispatcher mode byte (take-home)
//	Encode4(ii->p->nITCSN)   @0x5738dc  the ITC serial number of the listing
//
// The function takes nTI/nPos args but does NOT write them to the wire — only
// nITCSN is sent; the server resolves the destination slot.
//
// packet-audit:fname CITC::OnMoveITCPurchaseItemLtoS
type ItcOperationMoveLtoS struct {
	mode  byte
	itcSn uint32 // Encode4 ii->p->nITCSN
}

func NewItcOperationMoveLtoS(mode byte, itcSn uint32) ItcOperationMoveLtoS {
	return ItcOperationMoveLtoS{mode: mode, itcSn: itcSn}
}

func (m ItcOperationMoveLtoS) Mode() byte        { return m.mode }
func (m ItcOperationMoveLtoS) ItcSn() uint32     { return m.itcSn }
func (m ItcOperationMoveLtoS) Operation() string { return ItcOperationHandle }
func (m ItcOperationMoveLtoS) String() string {
	return fmt.Sprintf("itc take-home mode [%d] itcSn [%d]", m.mode, m.itcSn)
}

func (m ItcOperationMoveLtoS) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // Encode1(8u) @0x5738c8 mode byte
		w.WriteInt(m.itcSn) // Encode4 @0x5738dc nITCSN
		return w.Bytes()
	}
}

func (m *ItcOperationMoveLtoS) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itcSn = r.ReadUint32()
	}
}

// ItcOperationPlaceBid — the place-bid arm. The v95 send is INLINED into
// CITCBidAuctionDlg::OnButtonClicked @0x58eb50 (the nId==1 confirm-bid branch);
// there is no standalone CITC::OnBid function. After COutPacket(308) @0x58eda1
// the branch encodes, in order:
//
//	Encode1(0x13u)               @0x58edb4  dispatcher mode byte (place bid)
//	Encode4(m_pITCItem.p->nITCSN) @0x58edc7  the ITC serial number of the listing
//	Encode4(m_nMyBidPrice)       @0x58edd7  the player's bid base price
//	Encode4(m_nMyBidRange)       @0x58ede7  the player's bid increment/range
//
// A balance check (GetPriceWithCommision vs nexon cash, @0x58ec34) and a max-
// price guard (@0x58ec8b) gate the send; neither changes the wire shape.
//
// packet-audit:fname CITCBidAuctionDlg::OnButtonClicked
type ItcOperationPlaceBid struct {
	mode     byte
	itcSn    uint32 // Encode4 m_pITCItem.p->nITCSN
	bidPrice uint32 // Encode4 m_nMyBidPrice
	bidRange uint32 // Encode4 m_nMyBidRange
}

func NewItcOperationPlaceBid(mode byte, itcSn uint32, bidPrice uint32, bidRange uint32) ItcOperationPlaceBid {
	return ItcOperationPlaceBid{mode: mode, itcSn: itcSn, bidPrice: bidPrice, bidRange: bidRange}
}

func (m ItcOperationPlaceBid) Mode() byte        { return m.mode }
func (m ItcOperationPlaceBid) ItcSn() uint32     { return m.itcSn }
func (m ItcOperationPlaceBid) BidPrice() uint32  { return m.bidPrice }
func (m ItcOperationPlaceBid) BidRange() uint32  { return m.bidRange }
func (m ItcOperationPlaceBid) Operation() string { return ItcOperationHandle }
func (m ItcOperationPlaceBid) String() string {
	return fmt.Sprintf("itc place bid mode [%d] itcSn [%d] price [%d] range [%d]", m.mode, m.itcSn, m.bidPrice, m.bidRange)
}

func (m ItcOperationPlaceBid) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)    // Encode1(0x13u) @0x58edb4 mode byte
		w.WriteInt(m.itcSn)    // Encode4 @0x58edc7 nITCSN
		w.WriteInt(m.bidPrice) // Encode4 @0x58edd7 m_nMyBidPrice
		w.WriteInt(m.bidRange) // Encode4 @0x58ede7 m_nMyBidRange
		return w.Bytes()
	}
}

func (m *ItcOperationPlaceBid) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itcSn = r.ReadUint32()
		m.bidPrice = r.ReadUint32()
		m.bidRange = r.ReadUint32()
	}
}

// WISH-LIST / ZZIM (favorite) arms of the same ITC_OPERATION dispatcher,
// verified on gms_v95 (GMS_v95.0_U_DEVM.exe, IDA port 13340 — the symbol-rich
// PDB build exposes these as named CITC::On* functions). All but
// OnRegisterWishEntry are serial-only (Encode4(nITCSN)) like the buy/cancel
// arms; OnRegisterWishEntry carries a full wish-entry body. Each sends the
// dispatcher opcode COutPacket(308/0x134) then a leading Encode1(mode) byte,
// derived per-function below:
//
//	CITC::OnSetZzim @0x5733b0 — COutPacket(308) @0x5733e5, Encode1(9) @0x5733f8,
//	    Encode4(ii->p->nITCSN) @0x57340c. Mode 0x09 (add to wishlist/favorite).
//	CITC::OnBuyZzim @0x573450 — a YesNo confirm (StringPool 0x12D4) gates the
//	    send; COutPacket(308) @0x5734b7, Encode1(0x11) @0x5734ca,
//	    Encode4(ii->p->nITCSN) @0x5734de. Mode 0x11 (buy a favorited item).
//	CITC::OnDeleteZzim @0x573520 — COutPacket(308) @0x573555, Encode1(0xA)
//	    @0x573568, Encode4(ii->p->nITCSN) @0x57357c. Mode 0x0A (remove favorite).
//	CITC::OnViewWish @0x5735c0 — COutPacket(308) @0x5735f5, Encode1(0xB)
//	    @0x573608, Encode4(ii->p->nITCSN) @0x57361c. Mode 0x0B (view wish-list).
//	CITC::OnBuyWish @0x573660 — COutPacket(308) @0x573695, Encode1(0xC)
//	    @0x5736a8, Encode4(ii->p->nITCSN) @0x5736bc. Mode 0x0C (buy from wish).
//	CITC::OnCancelWish @0x573700 — COutPacket(308) @0x573735, Encode1(0xD)
//	    @0x573748, Encode4(ii->p->nITCSN) @0x57375c. Mode 0x0D (cancel a wish).
//	CITC::OnRegisterWishEntry @0x573c10 — COutPacket(308) @0x573ca5,
//	    Encode1(4) @0x573cb5, then the wish-entry body (see below). Mode 0x04.
//
// Each of these carries the same body shape across all versions (per-version
// opcode + mode bytes); v95 is the symbol-rich reference for propagation.

// ItcOperationSetZzim — the add-to-wishlist/favorite arm (CITC::OnSetZzim
// @0x5733b0, gms_v95). After COutPacket(308) @0x5733e5 it encodes, in order:
//
//	Encode1(9u)            @0x5733f8  dispatcher mode byte (set zzim)
//	Encode4(ii->p->nITCSN) @0x57340c  itcSn
//
// The m_bITCRequestSent latch (@0x5733d6) guards a double-send; not on the wire.
//
// packet-audit:fname CITC::OnSetZzim
type ItcOperationSetZzim struct {
	mode  byte
	itcSn uint32 // Encode4 ii->p->nITCSN
}

func NewItcOperationSetZzim(mode byte, itcSn uint32) ItcOperationSetZzim {
	return ItcOperationSetZzim{mode: mode, itcSn: itcSn}
}

func (m ItcOperationSetZzim) Mode() byte        { return m.mode }
func (m ItcOperationSetZzim) ItcSn() uint32     { return m.itcSn }
func (m ItcOperationSetZzim) Operation() string { return ItcOperationHandle }
func (m ItcOperationSetZzim) String() string {
	return fmt.Sprintf("itc set zzim mode [%d] itcSn [%d]", m.mode, m.itcSn)
}

func (m ItcOperationSetZzim) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // Encode1(9u) @0x5733f8 mode byte
		w.WriteInt(m.itcSn) // Encode4 @0x57340c nITCSN
		return w.Bytes()
	}
}

func (m *ItcOperationSetZzim) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itcSn = r.ReadUint32()
	}
}

// ItcOperationBuyZzim — the buy-a-favorited-item arm (CITC::OnBuyZzim
// @0x573450, gms_v95). A YesNo confirm (StringPool 0x12D4, @0x57349c) gates the
// send; the wire shape after COutPacket(308) @0x5734b7 is:
//
//	Encode1(0x11u)         @0x5734ca  dispatcher mode byte (buy zzim)
//	Encode4(ii->p->nITCSN) @0x5734de  itcSn
//
// packet-audit:fname CITC::OnBuyZzim
type ItcOperationBuyZzim struct {
	mode  byte
	itcSn uint32 // Encode4 ii->p->nITCSN
}

func NewItcOperationBuyZzim(mode byte, itcSn uint32) ItcOperationBuyZzim {
	return ItcOperationBuyZzim{mode: mode, itcSn: itcSn}
}

func (m ItcOperationBuyZzim) Mode() byte        { return m.mode }
func (m ItcOperationBuyZzim) ItcSn() uint32     { return m.itcSn }
func (m ItcOperationBuyZzim) Operation() string { return ItcOperationHandle }
func (m ItcOperationBuyZzim) String() string {
	return fmt.Sprintf("itc buy zzim mode [%d] itcSn [%d]", m.mode, m.itcSn)
}

func (m ItcOperationBuyZzim) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // Encode1(0x11u) @0x5734ca mode byte
		w.WriteInt(m.itcSn) // Encode4 @0x5734de nITCSN
		return w.Bytes()
	}
}

func (m *ItcOperationBuyZzim) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itcSn = r.ReadUint32()
	}
}

// ItcOperationDeleteZzim — the remove-favorite arm (CITC::OnDeleteZzim
// @0x573520, gms_v95). After COutPacket(308) @0x573555 it encodes, in order:
//
//	Encode1(0xAu)          @0x573568  dispatcher mode byte (delete zzim)
//	Encode4(ii->p->nITCSN) @0x57357c  itcSn
//
// packet-audit:fname CITC::OnDeleteZzim
type ItcOperationDeleteZzim struct {
	mode  byte
	itcSn uint32 // Encode4 ii->p->nITCSN
}

func NewItcOperationDeleteZzim(mode byte, itcSn uint32) ItcOperationDeleteZzim {
	return ItcOperationDeleteZzim{mode: mode, itcSn: itcSn}
}

func (m ItcOperationDeleteZzim) Mode() byte        { return m.mode }
func (m ItcOperationDeleteZzim) ItcSn() uint32     { return m.itcSn }
func (m ItcOperationDeleteZzim) Operation() string { return ItcOperationHandle }
func (m ItcOperationDeleteZzim) String() string {
	return fmt.Sprintf("itc delete zzim mode [%d] itcSn [%d]", m.mode, m.itcSn)
}

func (m ItcOperationDeleteZzim) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // Encode1(0xAu) @0x573568 mode byte
		w.WriteInt(m.itcSn) // Encode4 @0x57357c nITCSN
		return w.Bytes()
	}
}

func (m *ItcOperationDeleteZzim) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itcSn = r.ReadUint32()
	}
}

// ItcOperationViewWish — the view-wish-list arm (CITC::OnViewWish @0x5735c0,
// gms_v95). After COutPacket(308) @0x5735f5 it encodes, in order:
//
//	Encode1(0xBu)          @0x573608  dispatcher mode byte (view wish)
//	Encode4(ii->p->nITCSN) @0x57361c  itcSn
//
// packet-audit:fname CITC::OnViewWish
type ItcOperationViewWish struct {
	mode  byte
	itcSn uint32 // Encode4 ii->p->nITCSN
}

func NewItcOperationViewWish(mode byte, itcSn uint32) ItcOperationViewWish {
	return ItcOperationViewWish{mode: mode, itcSn: itcSn}
}

func (m ItcOperationViewWish) Mode() byte        { return m.mode }
func (m ItcOperationViewWish) ItcSn() uint32     { return m.itcSn }
func (m ItcOperationViewWish) Operation() string { return ItcOperationHandle }
func (m ItcOperationViewWish) String() string {
	return fmt.Sprintf("itc view wish mode [%d] itcSn [%d]", m.mode, m.itcSn)
}

func (m ItcOperationViewWish) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // Encode1(0xBu) @0x573608 mode byte
		w.WriteInt(m.itcSn) // Encode4 @0x57361c nITCSN
		return w.Bytes()
	}
}

func (m *ItcOperationViewWish) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itcSn = r.ReadUint32()
	}
}

// ItcOperationBuyWish — the buy-from-wish arm (CITC::OnBuyWish @0x573660,
// gms_v95). After COutPacket(308) @0x573695 it encodes, in order:
//
//	Encode1(0xCu)          @0x5736a8  dispatcher mode byte (buy wish)
//	Encode4(ii->p->nITCSN) @0x5736bc  itcSn
//
// packet-audit:fname CITC::OnBuyWish
type ItcOperationBuyWish struct {
	mode  byte
	itcSn uint32 // Encode4 ii->p->nITCSN
}

func NewItcOperationBuyWish(mode byte, itcSn uint32) ItcOperationBuyWish {
	return ItcOperationBuyWish{mode: mode, itcSn: itcSn}
}

func (m ItcOperationBuyWish) Mode() byte        { return m.mode }
func (m ItcOperationBuyWish) ItcSn() uint32     { return m.itcSn }
func (m ItcOperationBuyWish) Operation() string { return ItcOperationHandle }
func (m ItcOperationBuyWish) String() string {
	return fmt.Sprintf("itc buy wish mode [%d] itcSn [%d]", m.mode, m.itcSn)
}

func (m ItcOperationBuyWish) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // Encode1(0xCu) @0x5736a8 mode byte
		w.WriteInt(m.itcSn) // Encode4 @0x5736bc nITCSN
		return w.Bytes()
	}
}

func (m *ItcOperationBuyWish) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itcSn = r.ReadUint32()
	}
}

// ItcOperationCancelWish — the cancel-a-wish arm (CITC::OnCancelWish
// @0x573700, gms_v95). After COutPacket(308) @0x573735 it encodes, in order:
//
//	Encode1(0xDu)          @0x573748  dispatcher mode byte (cancel wish)
//	Encode4(ii->p->nITCSN) @0x57375c  itcSn
//
// packet-audit:fname CITC::OnCancelWish
type ItcOperationCancelWish struct {
	mode  byte
	itcSn uint32 // Encode4 ii->p->nITCSN
}

func NewItcOperationCancelWish(mode byte, itcSn uint32) ItcOperationCancelWish {
	return ItcOperationCancelWish{mode: mode, itcSn: itcSn}
}

func (m ItcOperationCancelWish) Mode() byte        { return m.mode }
func (m ItcOperationCancelWish) ItcSn() uint32     { return m.itcSn }
func (m ItcOperationCancelWish) Operation() string { return ItcOperationHandle }
func (m ItcOperationCancelWish) String() string {
	return fmt.Sprintf("itc cancel wish mode [%d] itcSn [%d]", m.mode, m.itcSn)
}

func (m ItcOperationCancelWish) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // Encode1(0xDu) @0x573748 mode byte
		w.WriteInt(m.itcSn) // Encode4 @0x57375c nITCSN
		return w.Bytes()
	}
}

func (m *ItcOperationCancelWish) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itcSn = r.ReadUint32()
	}
}

// ItcOperationRegisterWishEntry — the register-a-wish-entry arm
// (CITC::OnRegisterWishEntry @0x573c10, gms_v95). Unlike the other wish/zzim
// arms this carries a full body. A 110-NX floor guard (nWishPrice-110, @0x573c56)
// gates a StringPool notice; it does not change the wire shape. After
// COutPacket(308) @0x573ca5 it encodes, in order:
//
//	Encode1(4u)                          @0x573cb5  dispatcher mode byte (register wish)
//	Encode4(m_nWishItemID)               @0x573cc5  wish item id
//	Encode4(m_nWishPrice)                @0x573cd5  wish price
//	Encode4(m_nWishCount)                @0x573ce5  wish count
//	Encode1(m_bWishDuration)             @0x573cf6  wish duration
//	Encode1(m_bWishRegistrationFeeOption)@0x573d07  registration-fee option
//	EncodeStr(m_sWishDesc)               @0x573d23  wish description (len-prefixed)
//
// packet-audit:fname CITC::OnRegisterWishEntry
type ItcOperationRegisterWishEntry struct {
	mode        byte
	itemId      uint32 // Encode4 m_nWishItemID
	price       uint32 // Encode4 m_nWishPrice
	count       uint32 // Encode4 m_nWishCount
	duration    byte   // Encode1 m_bWishDuration
	feeOption   byte   // Encode1 m_bWishRegistrationFeeOption
	description string // EncodeStr m_sWishDesc
}

func NewItcOperationRegisterWishEntry(mode byte, itemId uint32, price uint32, count uint32, duration byte, feeOption byte, description string) ItcOperationRegisterWishEntry {
	return ItcOperationRegisterWishEntry{mode: mode, itemId: itemId, price: price, count: count, duration: duration, feeOption: feeOption, description: description}
}

func (m ItcOperationRegisterWishEntry) Mode() byte          { return m.mode }
func (m ItcOperationRegisterWishEntry) ItemId() uint32      { return m.itemId }
func (m ItcOperationRegisterWishEntry) Price() uint32       { return m.price }
func (m ItcOperationRegisterWishEntry) Count() uint32       { return m.count }
func (m ItcOperationRegisterWishEntry) Duration() byte      { return m.duration }
func (m ItcOperationRegisterWishEntry) FeeOption() byte     { return m.feeOption }
func (m ItcOperationRegisterWishEntry) Description() string { return m.description }
func (m ItcOperationRegisterWishEntry) Operation() string   { return ItcOperationHandle }
func (m ItcOperationRegisterWishEntry) String() string {
	return fmt.Sprintf("itc register wish entry mode [%d] item [%d] price [%d] count [%d] duration [%d] fee [%d]", m.mode, m.itemId, m.price, m.count, m.duration, m.feeOption)
}

func (m ItcOperationRegisterWishEntry) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)               // Encode1(4u) @0x573cb5 mode byte
		w.WriteInt(m.itemId)              // Encode4 @0x573cc5 wish item id
		w.WriteInt(m.price)               // Encode4 @0x573cd5 wish price
		w.WriteInt(m.count)               // Encode4 @0x573ce5 wish count
		w.WriteByte(m.duration)           // Encode1 @0x573cf6 wish duration
		w.WriteByte(m.feeOption)          // Encode1 @0x573d07 registration-fee option
		w.WriteAsciiString(m.description) // EncodeStr @0x573d23 wish description
		return w.Bytes()
	}
}

func (m *ItcOperationRegisterWishEntry) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itemId = r.ReadUint32()
		m.price = r.ReadUint32()
		m.count = r.ReadUint32()
		m.duration = r.ReadByte()
		m.feeOption = r.ReadByte()
		m.description = r.ReadAsciiString()
	}
}

// BROWSE-NAVIGATION arms of the ITC_OPERATION dispatcher, verified on gms_v95
// (GMS_v95.0_U_DEVM.exe, IDA port 13340). Three sender functions
// (CITC::OnChangedCategory / OnChangedCategorySub / OnChangedPage) all emit the
// SAME mode byte 0x05 (the GetItcList browse request) with the same 8-field
// wire shape; they differ only in which fields they fill from args vs CITC
// state. A fourth (CITCWnd_Tab::OnButtonClicked, the tab/search-by-name button)
// emits a DIFFERENT mode byte 0x06 with a 6-field shape (no sort bytes). Per the
// dispatcher-family rule each sender is modeled by its own discrete struct that
// fixes the constant fills its decompile shows:
//
//	CITC::OnChangedCategory @0x5744a0 — COutPacket(308) @0x57451a, Encode1(5)
//	    @0x57452d. Sends nCategory then constant sub=0/page=0/sort=1/1/opt=1/"".
//	CITC::OnChangedCategorySub @0x5739a0 — COutPacket(308) @0x5739da, Encode1(5)
//	    @0x5739ed. cat=m_nCurCategory, sub=nCategorySub, page=0(const), the two
//	    sort bytes + searchOption + searchCondition from state/args.
//	CITC::OnChangedPage @0x573af0 — COutPacket(308) @0x573b29, Encode1(5)
//	    @0x573b3c. All 8 fields from CITC state (page=nPage).
//	CITCWnd_Tab::OnButtonClicked @0x584b10 — COutPacket(308) @0x584bc7/@0x584cc9,
//	    Encode1(6) @0x584bd7/@0x584cd9. cat=m_nSelect+1, sub=m_nSelect, page=0
//	    (const), searchOption, searchName. No sortType/sortColumn bytes.

// ItcOperationChangedCategory — the change-browse-category arm
// (CITC::OnChangedCategory @0x5744a0, gms_v95). After COutPacket(308) @0x57451a
// it encodes, in order (cited to the decompile):
//
//	Encode1(5u)             @0x57452d  dispatcher mode byte (GetItcList browse)
//	Encode4(nCategory)      @0x57453b  category (the only variable field)
//	Encode4(0)              @0x574546  categorySub (const 0)
//	Encode4(0)              @0x574551  page (const 0)
//	Encode1(1u)             @0x57455c  sortType (const 1)
//	Encode1(1u)             @0x574567  sortColumn (const 1)
//	Encode4(1u)             @0x574572  searchOption (const 1)
//	EncodeStr("")           @0x5745ac  searchCondition (const empty)
//
// SetSearchCondition(0,0,0,0,"") @0x57450c resets browse state before the send;
// it does not change the wire. The m_bITCRequestSent latch (@0x5744c7) guards a
// double-send; not on the wire.
//
// packet-audit:fname CITC::OnChangedCategory
type ItcOperationChangedCategory struct {
	mode     byte
	category uint32 // Encode4 nCategory
}

func NewItcOperationChangedCategory(mode byte, category uint32) ItcOperationChangedCategory {
	return ItcOperationChangedCategory{mode: mode, category: category}
}

func (m ItcOperationChangedCategory) Mode() byte        { return m.mode }
func (m ItcOperationChangedCategory) Category() uint32  { return m.category }
func (m ItcOperationChangedCategory) Operation() string { return ItcOperationHandle }
func (m ItcOperationChangedCategory) String() string {
	return fmt.Sprintf("itc changed category mode [%d] category [%d]", m.mode, m.category)
}

func (m ItcOperationChangedCategory) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)    // Encode1(5u) @0x57452d mode byte
		w.WriteInt(m.category) // Encode4 @0x57453b category
		w.WriteInt(0)          // Encode4(0) @0x574546 categorySub
		w.WriteInt(0)          // Encode4(0) @0x574551 page
		w.WriteByte(0x01)      // Encode1(1u) @0x57455c sortType
		w.WriteByte(0x01)      // Encode1(1u) @0x574567 sortColumn
		w.WriteInt(1)          // Encode4(1u) @0x574572 searchOption
		w.WriteAsciiString("") // EncodeStr("") @0x5745ac searchCondition
		return w.Bytes()
	}
}

func (m *ItcOperationChangedCategory) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.category = r.ReadUint32()
		_ = r.ReadUint32() // categorySub (const 0)
		_ = r.ReadUint32() // page (const 0)
		_ = r.ReadByte()   // sortType (const 1)
		_ = r.ReadByte()   // sortColumn (const 1)
		_ = r.ReadUint32() // searchOption (const 1)
		_ = r.ReadAsciiString()
	}
}

// ItcOperationChangedCategorySub — the change-sub-category arm
// (CITC::OnChangedCategorySub @0x5739a0, gms_v95). Mode 0x05, full 8-field
// browse-request body. After COutPacket(308) @0x5739da it encodes, in order:
//
//	Encode1(5u)                  @0x5739ed  dispatcher mode byte
//	Encode4(this->m_nCurCategory)@0x5739fa  category
//	Encode4(nCategorySub)        @0x573a08  categorySub
//	Encode4(0)                   @0x573a13  page (const 0)
//	Encode1(nSortType)           @0x573a21  sortType
//	Encode1(nSortColumn)         @0x573a2f  sortColumn
//	Encode4(searchOption)        @0x573a3f/0x573a7e  searchOption (1 or m_nSearchOption)
//	EncodeStr(searchCondition)   @0x573aa0  searchCondition
//
// The searchOption/condition branch (m_nCurCategorySub==nCategorySub) selects
// state vs the (1, "") reset; both branches keep the same wire shape.
//
// packet-audit:fname CITC::OnChangedCategorySub
type ItcOperationChangedCategorySub struct {
	mode            byte
	category        uint32 // Encode4 m_nCurCategory
	categorySub     uint32 // Encode4 nCategorySub
	sortType        byte   // Encode1 nSortType
	sortColumn      byte   // Encode1 nSortColumn
	searchOption    uint32 // Encode4 searchOption
	searchCondition string // EncodeStr searchCondition
}

func NewItcOperationChangedCategorySub(mode byte, category uint32, categorySub uint32, sortType byte, sortColumn byte, searchOption uint32, searchCondition string) ItcOperationChangedCategorySub {
	return ItcOperationChangedCategorySub{mode: mode, category: category, categorySub: categorySub, sortType: sortType, sortColumn: sortColumn, searchOption: searchOption, searchCondition: searchCondition}
}

func (m ItcOperationChangedCategorySub) Mode() byte              { return m.mode }
func (m ItcOperationChangedCategorySub) Category() uint32        { return m.category }
func (m ItcOperationChangedCategorySub) CategorySub() uint32     { return m.categorySub }
func (m ItcOperationChangedCategorySub) SortType() byte          { return m.sortType }
func (m ItcOperationChangedCategorySub) SortColumn() byte        { return m.sortColumn }
func (m ItcOperationChangedCategorySub) SearchOption() uint32    { return m.searchOption }
func (m ItcOperationChangedCategorySub) SearchCondition() string { return m.searchCondition }
func (m ItcOperationChangedCategorySub) Operation() string       { return ItcOperationHandle }
func (m ItcOperationChangedCategorySub) String() string {
	return fmt.Sprintf("itc changed category-sub mode [%d] category [%d] sub [%d] sort [%d/%d] option [%d]", m.mode, m.category, m.categorySub, m.sortType, m.sortColumn, m.searchOption)
}

func (m ItcOperationChangedCategorySub) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)                   // Encode1(5u) @0x5739ed mode byte
		w.WriteInt(m.category)                // Encode4 @0x5739fa category
		w.WriteInt(m.categorySub)             // Encode4 @0x573a08 categorySub
		w.WriteInt(0)                         // Encode4(0) @0x573a13 page
		w.WriteByte(m.sortType)               // Encode1 @0x573a21 sortType
		w.WriteByte(m.sortColumn)             // Encode1 @0x573a2f sortColumn
		w.WriteInt(m.searchOption)            // Encode4 @0x573a3f/0x573a7e searchOption
		w.WriteAsciiString(m.searchCondition) // EncodeStr @0x573aa0 searchCondition
		return w.Bytes()
	}
}

func (m *ItcOperationChangedCategorySub) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.category = r.ReadUint32()
		m.categorySub = r.ReadUint32()
		_ = r.ReadUint32() // page (const 0)
		m.sortType = r.ReadByte()
		m.sortColumn = r.ReadByte()
		m.searchOption = r.ReadUint32()
		m.searchCondition = r.ReadAsciiString()
	}
}

// ItcOperationChangedPage — the change-browse-page arm (CITC::OnChangedPage
// @0x573af0, gms_v95). Mode 0x05, full 8-field browse-request body, every field
// from CITC state. After COutPacket(308) @0x573b29 it encodes, in order:
//
//	Encode1(5u)                     @0x573b3c  dispatcher mode byte
//	Encode4(this->m_nCurCategory)   @0x573b49  category
//	Encode4(this->m_nCurCategorySub)@0x573b56  categorySub
//	Encode4(nPage)                  @0x573b64  page
//	Encode1(this->m_nSortType)      @0x573b72  sortType
//	Encode1(this->m_nSortColumn)    @0x573b80  sortColumn
//	Encode4(this->m_nSearchOption)  @0x573b90  searchOption
//	EncodeStr(this->m_sSearchCondition) @0x573bb2  searchCondition
//
// packet-audit:fname CITC::OnChangedPage
type ItcOperationChangedPage struct {
	mode            byte
	category        uint32 // Encode4 m_nCurCategory
	categorySub     uint32 // Encode4 m_nCurCategorySub
	page            uint32 // Encode4 nPage
	sortType        byte   // Encode1 m_nSortType
	sortColumn      byte   // Encode1 m_nSortColumn
	searchOption    uint32 // Encode4 m_nSearchOption
	searchCondition string // EncodeStr m_sSearchCondition
}

func NewItcOperationChangedPage(mode byte, category uint32, categorySub uint32, page uint32, sortType byte, sortColumn byte, searchOption uint32, searchCondition string) ItcOperationChangedPage {
	return ItcOperationChangedPage{mode: mode, category: category, categorySub: categorySub, page: page, sortType: sortType, sortColumn: sortColumn, searchOption: searchOption, searchCondition: searchCondition}
}

func (m ItcOperationChangedPage) Mode() byte              { return m.mode }
func (m ItcOperationChangedPage) Category() uint32        { return m.category }
func (m ItcOperationChangedPage) CategorySub() uint32     { return m.categorySub }
func (m ItcOperationChangedPage) Page() uint32            { return m.page }
func (m ItcOperationChangedPage) SortType() byte          { return m.sortType }
func (m ItcOperationChangedPage) SortColumn() byte        { return m.sortColumn }
func (m ItcOperationChangedPage) SearchOption() uint32    { return m.searchOption }
func (m ItcOperationChangedPage) SearchCondition() string { return m.searchCondition }
func (m ItcOperationChangedPage) Operation() string       { return ItcOperationHandle }
func (m ItcOperationChangedPage) String() string {
	return fmt.Sprintf("itc changed page mode [%d] category [%d] sub [%d] page [%d]", m.mode, m.category, m.categorySub, m.page)
}

func (m ItcOperationChangedPage) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)                   // Encode1(5u) @0x573b3c mode byte
		w.WriteInt(m.category)                // Encode4 @0x573b49 category
		w.WriteInt(m.categorySub)             // Encode4 @0x573b56 categorySub
		w.WriteInt(m.page)                    // Encode4 @0x573b64 page
		w.WriteByte(m.sortType)               // Encode1 @0x573b72 sortType
		w.WriteByte(m.sortColumn)             // Encode1 @0x573b80 sortColumn
		w.WriteInt(m.searchOption)            // Encode4 @0x573b90 searchOption
		w.WriteAsciiString(m.searchCondition) // EncodeStr @0x573bb2 searchCondition
		return w.Bytes()
	}
}

func (m *ItcOperationChangedPage) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.category = r.ReadUint32()
		m.categorySub = r.ReadUint32()
		m.page = r.ReadUint32()
		m.sortType = r.ReadByte()
		m.sortColumn = r.ReadByte()
		m.searchOption = r.ReadUint32()
		m.searchCondition = r.ReadAsciiString()
	}
}

// ItcOperationTabSearch — the tab/search-by-name browse request inlined into
// CITCWnd_Tab::OnButtonClicked @0x584b10 (gms_v95, the nId==1004 search-button
// branch; there is no standalone CITC::On* sender). It emits the DISTINCT mode
// byte 0x06 with a 6-field body (NO sortType/sortColumn bytes, unlike mode
// 0x05). Both inner branches (character-name search @0x584bd7, advanced-search
// @0x584cd9) send the identical wire shape. After COutPacket(308)
// @0x584bc7/@0x584cc9 it encodes, in order:
//
//	Encode1(6u)              @0x584bd7/@0x584cd9  dispatcher mode byte (search)
//	Encode4(m_nSelect+1)     @0x584be1/@0x584ce3  category (main-category select +1)
//	Encode4(m_nSelect)       @0x584beb/@0x584ced  categorySub (sub-category select)
//	Encode4(0)               @0x584bf5/@0x584cf8  page (const 0)
//	Encode4(searchOption)    @0x584bff/@0x584d02  searchOption (0 for name search, else m_nSelect)
//	EncodeStr(searchName)    @0x584c1b/@0x584d22  searchCondition (the edit-box text)
//
// packet-audit:fname CITCWnd_Tab::OnButtonClicked
type ItcOperationTabSearch struct {
	mode         byte
	category     uint32 // Encode4 m_nSelect+1
	categorySub  uint32 // Encode4 m_nSelect
	searchOption uint32 // Encode4 searchOption
	searchName   string // EncodeStr searchCondition
}

func NewItcOperationTabSearch(mode byte, category uint32, categorySub uint32, searchOption uint32, searchName string) ItcOperationTabSearch {
	return ItcOperationTabSearch{mode: mode, category: category, categorySub: categorySub, searchOption: searchOption, searchName: searchName}
}

func (m ItcOperationTabSearch) Mode() byte           { return m.mode }
func (m ItcOperationTabSearch) Category() uint32     { return m.category }
func (m ItcOperationTabSearch) CategorySub() uint32  { return m.categorySub }
func (m ItcOperationTabSearch) SearchOption() uint32 { return m.searchOption }
func (m ItcOperationTabSearch) SearchName() string   { return m.searchName }
func (m ItcOperationTabSearch) Operation() string    { return ItcOperationHandle }
func (m ItcOperationTabSearch) String() string {
	return fmt.Sprintf("itc tab search mode [%d] category [%d] sub [%d] option [%d] name [%s]", m.mode, m.category, m.categorySub, m.searchOption, m.searchName)
}

func (m ItcOperationTabSearch) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)              // Encode1(6u) @0x584bd7/@0x584cd9 mode byte
		w.WriteInt(m.category)           // Encode4 @0x584be1/@0x584ce3 category
		w.WriteInt(m.categorySub)        // Encode4 @0x584beb/@0x584ced categorySub
		w.WriteInt(0)                    // Encode4(0) @0x584bf5/@0x584cf8 page
		w.WriteInt(m.searchOption)       // Encode4 @0x584bff/@0x584d02 searchOption
		w.WriteAsciiString(m.searchName) // EncodeStr @0x584c1b/@0x584d22 searchCondition
		return w.Bytes()
	}
}

func (m *ItcOperationTabSearch) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.category = r.ReadUint32()
		m.categorySub = r.ReadUint32()
		_ = r.ReadUint32() // page (const 0)
		m.searchOption = r.ReadUint32()
		m.searchName = r.ReadAsciiString()
	}
}
