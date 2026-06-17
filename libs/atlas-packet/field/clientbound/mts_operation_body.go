package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// MtsOperationWriter is the registry writer name (Operation()) shared by every
// per-mode CITC::OnNormalItemResult (MTS_OPERATION) body codec below. The
// mode-only MtsOperation struct that originally declared this const was retired
// in task-096 once the per-mode body functions (field/operation_body.go) made it
// dead; the const survives here because the codecs use it as their Operation().
const MtsOperationWriter = "MtsOperation"

// Per-mode body codecs for the CITC::OnNormalItemResult dispatcher
// (MTS_OPERATION). The dispatcher reads Decode1(mode) and switch-dispatches to
// one of 35 sub-handlers (0x15..0x3E). The mode bytes are version-stable across
// gms_v83 (dispatcher 0x5a4311), gms_v84 (0x5b47c8), gms_v87 (0x5d43d0),
// gms_v95 (0x5771d0) — IDA-confirmed: the case labels (21..62) are identical and
// the sub-handler bodies are byte-identical in shape. jms_v185 has no CITC op
// (registry-absent).
//
// Each struct writes the leading mode byte THEN the full arm body (the read
// order the matching CITC sub-handler performs on CInPacket). This replaces the
// mode-byte-only MtsOperation false-pass for the arms whose body shape is
// scalar and fully decompile-cited below. The heavier list arms
// (Get*ListDone / Get*ItemDone / LoadWishSaleListDone) embed the ITCITEM /
// GW_ItemSlotBase item blob and are modeled in mts_operation_list.go (MtsItem).

// MtsResultEmpty — arms whose CITC sub-handler reads NOTHING from the packet
// beyond the mode byte already consumed by the dispatcher (it only shows a
// StringPool notice and clears m_bITCRequestSent). The codec writes the mode
// byte and stops — the verified wire contract for these arms.
//
// VERIFIED iteration 1 (task-096), per-version sub-handler addresses
// (CITC::OnNormalItemResult dispatcher: v83 0x5a4311 / v84 0x5b47c8 /
// v87 0x5d43d0 / v95 0x5771d0). Each sub-handler is a StringPool::GetString +
// CUtilDlg::Notice with NO CInPacket::Decode* after the dispatcher's Decode1:
//
//	0x1D RegisterSaleEntryDone   v83 0x5a4674 / v84 0x5b4b64 / v87 0x5d4748 / v95 0x575cd0
//	0x1F SaleCurrentItemToWishDone v83 0x5a46b2 / v84 0x5b4ba2 / v87 0x5d4786 / v95 0x575d20
//	0x29 SetZzimDone             v83 0x5a4dfc / v84 0x5b52ec / v87 0x5d4eef / v95 0x576140
//	0x2A SetZzimFailed           v83 0x5a4e31 / v84 0x5b5321 / v87 0x5d4f24 / v95 0x576180
//
// VERIFIED iteration 2 (task-096), six more Empty-shape arms, each decompiled in
// ALL FOUR versions and confirmed to be StringPool::GetString + CUtilDlg::Notice
// with NO CInPacket::Decode* after the dispatcher's Decode1 (the trailing
// this[6]/m_bITCRequestSent=0 on the *Failed/CancelSaleItemDone arms is a member
// store, not a wire read):
//
//	0x25 CancelSaleItemDone      v83 0x5a4d14 / v84 0x5b5204 / v87 0x5d4e04 / v95 0x576030
//	0x2B DeleteZzimDone          v83 0x5a4e66 / v84 0x5b5356 / v87 0x5d4f59 / v95 0x5761c0
//	0x2C DeleteZzimFailed        v83 0x5a4e91 / v84 0x5b5381 / v87 0x5d4f84 / v95 0x5761f0
//	0x2E LoadWishSaleListFailed  v83 0x5a4fdc / v84 0x5b54cc / v87 0x5d50cf / v95 0x576230
//	0x2F BuyWishDone             v83 0x5a5011 / v84 0x5b5501 / v87 0x5d5104 / v95 0x576270
//	0x30 BuyWishFailed           v83 0x5a503c / v84 0x5b552c / v87 0x5d512f / v95 0x5762a0
//
// (v84 sub-handler addresses are case = Decode1-21 in the v84 dispatcher
// sub_5B47C8; the iteration-2 v84 column was corrected in iteration 3 after
// re-deriving each case label.)
//
// VERIFIED iteration 3 (task-096), six more Empty-shape arms, each decompiled in
// ALL FOUR versions and confirmed StringPool::GetString + CUtilDlg::Notice with
// a trailing this[6]/m_bITCRequestSent=0 member store and NO CInPacket::Decode*
// after the dispatcher's Decode1:
//
//	0x31 CancelWishDone          v83 0x5a5071 / v84 0x5b5561 / v87 0x5d5164 / v95 0x5762e0
//	0x32 CancelWishFailed        v83 0x5a50df / v84 0x5b5596 / v87 0x5d5199 / v95 0x576320
//	0x34 BuyItemFailed           v83 0x5a513f / v84 0x5b55f6 / v87 0x5d51f9 / v95 0x576390
//	0x36 BuyZzimItemFailed       v83 0x5a519f / v84 0x5b5656 / v87 0x5d5259 / v95 0x576400
//	0x38 RegisterWishItemFailed  v83 0x5a5209 / v84 0x5b56c0 / v87 0x5d52c3 / v95 0x576480
//	0x3C BidAuctionFailed        v83 0x5a5444 / v84 0x5b58fb / v87 0x5d54fe / v95 0x5764c0
//
// VERIFIED iteration 4 (task-096), three more Empty-shape arms, each decompiled
// in ALL FOUR versions and confirmed StringPool::GetString + CUtilDlg::Notice
// (RegisterWishItemDone also stores this[6]/m_bITCRequestSent=0) with NO
// CInPacket::Decode* after the dispatcher's Decode1:
//
//	0x33 BuyItemDone           v83 0x5a5114 / v84 0x5b55cb / v87 0x5d51ce / v95 0x576360
//	0x35 BuyZzimItemDone       v83 0x5a5174 / v84 0x5b562b / v87 0x5d522e / v95 0x5763d0
//	0x37 RegisterWishItemDone  v83 0x5a51d4 / v84 0x5b568b / v87 0x5d528e / v95 0x576440
//
// packet-audit:fname CITC::OnNormalItemResult#Empty
type MtsResultEmpty struct {
	mode byte
}

func NewMtsResultEmpty(mode byte) MtsResultEmpty {
	return MtsResultEmpty{mode: mode}
}

func (m MtsResultEmpty) Mode() byte        { return m.mode }
func (m MtsResultEmpty) Operation() string { return MtsOperationWriter }
func (m MtsResultEmpty) String() string    { return fmt.Sprintf("mts result (notice) mode [%d]", m.mode) }

func (m MtsResultEmpty) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte; sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultEmpty) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultReason — arms whose CITC sub-handler reads a single Decode1
// fail-reason byte (then routes a StringPool notice via NoticeFailReason or a
// reason-keyed GetString). The codec writes the mode byte THEN the Decode1
// reason byte — the verified wire contract for these arms.
//
// VERIFIED this batch (task-096 iteration 1), per-version sub-handler addresses:
//
//	0x16 GetITCListFailed         v83 0x5a4882 / v84 0x5b4d72 / v87 0x5d4972 / v95 0x575f70
//	     (Decode1 reason -> NoticeFailReason; reason==73 also re-sends transfer)
//	0x20 SaleCurrentItemToWishFailed v83 0x5a46f0 / v84 0x5b4be0 / v87 0x5d47c4 / v95 0x575d70
//	     (Decode1 reason -> reason-keyed GetString; 80/82|83/else)
//
// Both read exactly Decode1(reason) on the wire after the dispatcher mode byte
// (the downstream StringPool selection uses the reason value but reads no more
// bytes).
//
// VERIFIED iteration 4 (task-096), three more Reason-shape arms, each decompiled
// in ALL FOUR versions and confirmed Decode1(reason) -> NoticeFailReason (the
// GetUser*Failed arms additionally re-send the transfer-field packet when
// reason==73, which reads NO further bytes):
//
//	0x18 GetSearchITCListFailed    v83 0x5a49e3 / v84 0x5b4ed3 / v87 0x5d4ad3 / v95 0x575fa0
//	0x22 GetUserPurchaseItemFailed v83 0x5a4c2a / v84 0x5b511a / v87 0x5d4d1a / v95 0x575fd0
//	0x24 GetUserSaleItemFailed     v83 0x5a4ce7 / v84 0x5b51d7 / v87 0x5d4dd7 / v95 0x576000
//
// VERIFIED iteration 5 (task-096), two more Reason-shape arms, each decompiled in
// ALL FOUR versions and confirmed Decode1(reason) -> NoticeFailReason with NO
// further CInPacket::Decode* (CancelSaleItemFailed reads exactly one byte;
// MoveITCPurchaseItemLtoSFailed's reason==65 branch re-sends the transfer-field
// packet, which reads no bytes):
//
//	0x26 CancelSaleItemFailed         v83 0x5a4d49 / v84 0x5b5239 / v87 0x5d4e39 / v95 0x576070
//	0x28 MoveITCPurchaseItemLtoSFailed v83 0x5a4dcf / v84 0x5b52bf / v87 0x5d4ec2 / v95 0x576110
//
// packet-audit:fname CITC::OnNormalItemResult#Reason
type MtsResultReason struct {
	mode   byte
	reason byte
}

func NewMtsResultReason(mode byte, reason byte) MtsResultReason {
	return MtsResultReason{mode: mode, reason: reason}
}

func (m MtsResultReason) Mode() byte        { return m.mode }
func (m MtsResultReason) Reason() byte      { return m.reason }
func (m MtsResultReason) Operation() string { return MtsOperationWriter }
func (m MtsResultReason) String() string {
	return fmt.Sprintf("mts result (fail) mode [%d] reason [%d]", m.mode, m.reason)
}

func (m MtsResultReason) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte
		w.WriteByte(m.reason) // Decode1 fail reason -> NoticeFailReason
		return w.Bytes()
	}
}

func (m *MtsResultReason) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
	}
}

// MtsResultTwoInts — arms whose sub-handler reads exactly two Decode4 ints after
// the dispatcher mode byte. Covers, VERIFIED iteration 5 in ALL FOUR versions
// (decompiled; identical Decode4 then Decode4 read order):
//
//	0x27 MoveITCPurchaseItemLtoSDone  v83 0x5a4d68 / v84 0x5b5258 / v87 0x5d4e58 / v95 0x5760a0
//	     (Decode4 v3 (tab+1 -> CCtrlTab::SetTab), Decode4 v4 (selectedNo))
//	0x3D NotifyCancelWishResult       v83 0x5a523e / v84 0x5b56f5 / v87 0x5d52f8 / v95 0x576f00
//	     (Decode4 v1 (count d), Decode4 v2 (count x); both >0 guards a notice)
//
// The sub-handler's downstream use differs (tab vs notice count) but the wire
// read order is identical: Decode4 then Decode4. The trailing this[6]=0 store on
// the LtoSDone arm is a member write, not a wire read.
//
// packet-audit:fname CITC::OnNormalItemResult#TwoInts
type MtsResultTwoInts struct {
	mode byte
	a    uint32
	b    uint32
}

func NewMtsResultTwoInts(mode byte, a uint32, b uint32) MtsResultTwoInts {
	return MtsResultTwoInts{mode: mode, a: a, b: b}
}

func (m MtsResultTwoInts) Mode() byte        { return m.mode }
func (m MtsResultTwoInts) Operation() string { return MtsOperationWriter }
func (m MtsResultTwoInts) String() string {
	return fmt.Sprintf("mts result (two ints) mode [%d] a [%d] b [%d]", m.mode, m.a, m.b)
}

func (m MtsResultTwoInts) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte
		w.WriteInt(m.a)     // Decode4
		w.WriteInt(m.b)     // Decode4
		return w.Bytes()
	}
}

func (m *MtsResultTwoInts) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.a = r.ReadUint32()
		m.b = r.ReadUint32()
	}
}

// mtsRegisterSaleEntryFailedSaleLimitReason is the only reason value of the
// 0x1E RegisterSaleEntryFailed arm whose sub-handler reads a trailing Decode2
// short (the per-account sale-limit count). All other reasons route a plain
// NoticeFailReason with no trailing read.
//
//	v83 sub_5A4581:  if ( v3 == 72 ) { v4 = CInPacket::Decode2(a2); ... }
//	v84 sub_5B4A38:  if ( v3 == 72 ) { v4 = CInPacket::Decode2(a2); ... }
//	v87 OnNormalItemResRegisterSaleEntryFailed@0x5d4640: if (v3==72) Decode2
//	v95 OnNormalItemResRegisterSaleEntryFailed@0x576b80: if (v3==72) Decode2
const mtsRegisterSaleEntryFailedSaleLimitReason byte = 0x48 // 72

// MtsResultRegisterSaleEntryFailed — the 0x1E RegisterSaleEntryFailed arm
// (CITC::OnNormalItemResRegisterSaleEntryFailed). The sub-handler reads
// Decode1(reason) -> NoticeFailReason, EXCEPT when reason==0x48 (sale-limit
// reached) where it then reads a single Decode2 short (the limit count, used to
// Format the StringPool 0x12BB notice). The codec writes the mode byte, the
// Decode1 reason byte, and — ONLY when reason==0x48 — the Decode2 limit short.
//
// VERIFIED iteration 5 (task-096), decompiled in ALL FOUR versions (identical
// gate + read order; the trailing m_bITCRequestSent=0 store is not a wire read):
//
//	v83 sub_5A4581          / v84 sub_5B4A38
//	v87 0x5d4640            / v95 0x576b80
//
// packet-audit:fname CITC::OnNormalItemResult#RegisterSaleEntryFailed
type MtsResultRegisterSaleEntryFailed struct {
	mode      byte
	reason    byte
	saleLimit uint16 // Decode2; present on the wire only when reason==0x48
}

func NewMtsResultRegisterSaleEntryFailed(reason byte, saleLimit uint16) MtsResultRegisterSaleEntryFailed {
	return MtsResultRegisterSaleEntryFailed{mode: 0x1E, reason: reason, saleLimit: saleLimit}
}

func (m MtsResultRegisterSaleEntryFailed) Mode() byte        { return m.mode }
func (m MtsResultRegisterSaleEntryFailed) Reason() byte      { return m.reason }
func (m MtsResultRegisterSaleEntryFailed) SaleLimit() uint16 { return m.saleLimit }
func (m MtsResultRegisterSaleEntryFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultRegisterSaleEntryFailed) String() string {
	return fmt.Sprintf("mts register sale entry failed mode [%d] reason [%d] saleLimit [%d]", m.mode, m.reason, m.saleLimit)
}

func (m MtsResultRegisterSaleEntryFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte (0x1E)
		w.WriteByte(m.reason) // Decode1 fail reason -> NoticeFailReason
		if m.reason == mtsRegisterSaleEntryFailedSaleLimitReason {
			w.WriteShort(m.saleLimit) // Decode2 sale-limit count (reason==0x48 branch only)
		}
		return w.Bytes()
	}
}

func (m *MtsResultRegisterSaleEntryFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
		if m.reason == mtsRegisterSaleEntryFailedSaleLimitReason {
			m.saleLimit = r.ReadUint16()
		}
	}
}

// MtsResultSuccessBidInfo — the 0x3E SuccessBidInfoResult arm
// (CITC::OnSuccessBidInfoResult). The sub-handler reads Decode1(soldFlag) +
// Decode4(itemId); when itemId>0 it then reads Decode4(price) and an 8-byte
// FILETIME contract date via DecodeBuffer(8). When itemId<=0 the body ends after
// the leading byte+int (no notice shown). The codec writes the mode byte, the
// Decode1 soldFlag, the Decode4 itemId, and — ONLY when itemId>0 — the Decode4
// price followed by the 8-byte contract-date buffer.
//
// VERIFIED iteration 5 (task-096), decompiled in ALL FOUR versions (identical
// read order: Decode1, Decode4, [itemId>0: Decode4 + DecodeBuffer(8)]):
//
//	v83 sub_5A52DE          / v84 sub_5B5795
//	v87 OnSuccessBidInfoResult@0x5d5398 / v95 OnSuccessBidInfoResult@0x577000
//
// packet-audit:fname CITC::OnNormalItemResult#SuccessBidInfo
type MtsResultSuccessBidInfo struct {
	mode         byte
	soldFlag     byte    // Decode1: 1=sold (StringPool 0x12AA), else bought (0x12AB)
	itemId       uint32  // Decode4: ITC item id; <=0 ends the body
	price        uint32  // Decode4: meso price (itemId>0 only)
	contractDate [8]byte // DecodeBuffer(8): FILETIME (itemId>0 only)
}

func NewMtsResultSuccessBidInfo(soldFlag byte, itemId uint32, price uint32, contractDate [8]byte) MtsResultSuccessBidInfo {
	return MtsResultSuccessBidInfo{mode: 0x3E, soldFlag: soldFlag, itemId: itemId, price: price, contractDate: contractDate}
}

func (m MtsResultSuccessBidInfo) Mode() byte            { return m.mode }
func (m MtsResultSuccessBidInfo) SoldFlag() byte        { return m.soldFlag }
func (m MtsResultSuccessBidInfo) ItemId() uint32        { return m.itemId }
func (m MtsResultSuccessBidInfo) Price() uint32         { return m.price }
func (m MtsResultSuccessBidInfo) ContractDate() [8]byte { return m.contractDate }
func (m MtsResultSuccessBidInfo) Operation() string     { return MtsOperationWriter }
func (m MtsResultSuccessBidInfo) String() string {
	return fmt.Sprintf("mts success bid info mode [%d] soldFlag [%d] itemId [%d] price [%d]", m.mode, m.soldFlag, m.itemId, m.price)
}

func (m MtsResultSuccessBidInfo) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)     // dispatcher mode byte (0x3E)
		w.WriteByte(m.soldFlag) // Decode1 sold/bought flag
		w.WriteInt(m.itemId)    // Decode4 ITC item id
		if m.itemId > 0 {
			w.WriteInt(m.price)                 // Decode4 meso price (itemId>0 branch)
			w.WriteByteArray(m.contractDate[:]) // DecodeBuffer(8) FILETIME contract date
		}
		return w.Bytes()
	}
}

func (m *MtsResultSuccessBidInfo) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.soldFlag = r.ReadByte()
		m.itemId = r.ReadUint32()
		if m.itemId > 0 {
			m.price = r.ReadUint32()
			copy(m.contractDate[:], r.ReadBytes(8))
		}
	}
}
