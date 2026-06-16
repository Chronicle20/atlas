package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

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
// GW_ItemSlotBase item blob and are not yet modeled here.

// MtsResultEmpty — arms whose CITC sub-handler reads NOTHING from the packet
// beyond the mode byte already consumed by the dispatcher (it only shows a
// StringPool notice and clears m_bITCRequestSent). The codec writes the mode
// byte and stops — the verified wire contract for these arms.
//
// VERIFIED this batch (task-096 iteration 1), per-version sub-handler addresses
// (CITC::OnNormalItemResult dispatcher: v83 0x5a4311 / v84 0x5b47c8 /
// v87 0x5d43d0 / v95 0x5771d0). Each sub-handler is a StringPool::GetString +
// CUtilDlg::Notice with NO CInPacket::Decode* after the dispatcher's Decode1:
//
//	0x1D RegisterSaleEntryDone   v83 0x5a4674 / v84 0x5b4b64 / v87 0x5d4748 / v95 0x575cd0
//	0x1F SaleCurrentItemToWishDone v83 0x5a46b2 / v84 0x5b4ba2 / v87 0x5d4786 / v95 0x575d20
//	0x29 SetZzimDone             v83 0x5a4dfc / v84 0x5b52ec / v87 0x5d4eef / v95 0x576140
//	0x2A SetZzimFailed           v83 0x5a4e31 / v84 0x5b5321 / v87 0x5d4f24 / v95 0x576180
//
// Additional Empty-shape arms decompile-confirmed in gms_v95 but not yet pinned
// (later iterations): 0x25 CancelSaleItemDone @0x576030, 0x2B DeleteZzimDone
// @0x5761c0, 0x2C DeleteZzimFailed @0x5761f0, 0x2E LoadWishSaleListFailed
// @0x576230, 0x2F BuyWishDone @0x576270, 0x30 BuyWishFailed @0x5762a0,
// 0x31 CancelWishDone @0x5762e0, 0x32 CancelWishFailed @0x576320,
// 0x33 BuyItemDone @0x576360, 0x34 BuyItemFailed @0x576390, 0x35 BuyZzimItemDone
// @0x5763d0, 0x36 BuyZzimItemFailed @0x576400, 0x37 RegisterWishItemDone
// @0x576440, 0x38 RegisterWishItemFailed @0x576480, 0x3C BidAuctionFailed @0x5764c0.
//
// packet-audit:fname CITC::OnNormalItemResult#Empty  (dispatcher family — see docs/packets/evidence/families.yaml)
type MtsResultEmpty struct {
	mode byte
}

func NewMtsResultEmpty(mode byte) MtsResultEmpty {
	return MtsResultEmpty{mode: mode}
}

func (m MtsResultEmpty) Mode() byte          { return m.mode }
func (m MtsResultEmpty) Operation() string   { return MtsOperationWriter }
func (m MtsResultEmpty) String() string      { return fmt.Sprintf("mts result (notice) mode [%d]", m.mode) }

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
// bytes). Additional Reason-shape arms decompile-confirmed in gms_v95 but not
// yet pinned (later iterations): 0x18 GetSearchITCListFailed @0x575fa0,
// 0x22 GetUserPurchaseItemFailed @0x575fd0, 0x24 GetUserSaleItemFailed @0x576000,
// 0x26 CancelSaleItemFailed @0x576070, 0x28 MoveITCPurchaseItemLtoSFailed @0x576110.
//
// packet-audit:fname CITC::OnNormalItemResult#Reason  (dispatcher family — see docs/packets/evidence/families.yaml)
type MtsResultReason struct {
	mode   byte
	reason byte
}

func NewMtsResultReason(mode byte, reason byte) MtsResultReason {
	return MtsResultReason{mode: mode, reason: reason}
}

func (m MtsResultReason) Mode() byte        { return m.mode }
func (m MtsResultReason) Reason() byte       { return m.reason }
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

// MtsResultTwoInts — arms whose sub-handler reads two Decode4 ints. Covers,
// decompile-confirmed in gms_v95:
//
//	0x27 MoveITCPurchaseItemLtoSDone (CITC::OnMoveITCPurchaseItemLtoSDone @0x5760a0 —
//	     Decode4 v3 (tab+1), Decode4 v4 (selectedNo))
//	0x3D NotifyCancelWishResult      (CITC::OnNotifyCancelWishResult @0x576f00 —
//	     Decode4 v3 (count d), Decode4 v5 (count x))
//
// The sub-handler's downstream use differs (tab vs notice count) but the wire
// read order is identical: Decode4 then Decode4. Version-stable: v83
// OnMoveITCPurchaseItemLtoSDone @0x5a4d68 and sub_5A523E both read Decode4×2.
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
