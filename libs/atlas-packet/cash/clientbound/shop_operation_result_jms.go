package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// JMS-only arms (task-183 follow-up). These 10 arms exist ONLY in the JMS v185
// DEVM switch (CCashShop::OnCashItemResult @ 0x48b5a5 — raw CInPacket::Decode1
// switch; verified 2026-07). Each targets a `sub_XXXX` with NO client symbol and
// NO GMS/legacy-canonical equivalent (the GMS/legacy switches were fully
// enumerated in task-183 and are entirely named; none of these map to a named
// GMS handler). The IDB names below are BEHAVIOR-DERIVED (JMS DEVM build
// stripped; no PDB) and cite the handler address + the decompile evidence — see
// docs/tasks/task-183-cashshop-result-family/arm-catalog.md "JMS-only arms"
// section. Every arm is a DISCRETE struct per INV-1, even the bodyless
// canned-notice arms.

// ---- mode 76: reason-notice (CCashShop__OnCashItemResShowGiftResultNotice @ 0x48ba24) ----

// GiftResultNotice - GIFT_RESULT_NOTICE arm. Handler reads one reason byte
// (Decode1) and passes it to the gift-result notice function sub_48F0F2
// (0x48f0f2), which maps 214/215/216 → StringPool notice ids 625/626/627
// (else 624). No further wire read. Wire: mode + reason byte. The reason is a
// NoticeFailReason-class UI reason code → the body func resolves it from the
// writer "errors" table (sibling failure-arm pattern), so the struct field is a
// plain byte matching LoadInventoryFailure.errorCode.
// packet-audit:fname CCashShop::OnCashItemResult#GIFT_RESULT_NOTICE
type GiftResultNotice struct {
	mode      byte
	errorCode byte
}

func NewGiftResultNotice(mode byte, errorCode byte) GiftResultNotice {
	return GiftResultNotice{mode: mode, errorCode: errorCode}
}

func (m GiftResultNotice) Mode() byte        { return m.mode }
func (m GiftResultNotice) ErrorCode() byte   { return m.errorCode }
func (m GiftResultNotice) Operation() string { return CashShopOperationWriter }

func (m GiftResultNotice) String() string {
	return fmt.Sprintf("cash gift-result notice mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m GiftResultNotice) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *GiftResultNotice) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// ---- mode 77: received-gift list (CCashShop__OnCashItemResLoadReceivedGiftDone @ 0x48ba3f) ----

// ReceivedGiftEntry is the per-record wire shape of the received-gift list —
// one DecodeBuffer(0xB0 = 176 bytes) blob per entry (disasm 0x48ba88 pushes
// 0xB0). It is NOT the 98-byte GW_GiftList / GiftListEntry used by
// LOAD_GIFT_SUCCESS. This JMS record has no named type in the stripped DEVM
// IDB; the fields below cite the exact byte offsets the handler reads
// (sub_48BA3F disasm 0x48ba9b..0x48bbca), and the head + inter-field gaps are
// reserved regions (not read by the handler) so Encode always emits exactly
// 176 bytes.
//
// Layout (byte offsets into the 176-byte record):
//
//	0x00-0x0B  reserved head (12 bytes) — not accessed by the handler
//	0x0C  itemId   int32   — mov edi,[eax+0Ch]; arg1 to gift-display helpers (sub_869D96 / sub_86A57C), paired with itemName
//	0x10  data1    int32   — mov ecx,[eax+10h]  (helper arg2; semantics unresolved in stripped build)
//	0x14  data2    int32   — mov ecx,[eax+14h]  (helper arg3)
//	0x18  giftType int32   — cmp [eax+18h],0; ==0 → points/coupon MapleTip path, !=0 → gift-notice path
//	0x1C  text     string  — Assign(p+0x1C) null-terminated within the 101-byte region 0x1C-0x80
//	0x81  sender   string  — Assign(p+0x81) null-terminated within the 33-byte region 0x81-0xA1
//	0xA2  itemName string  — Assign(p+0xA2) null-terminated within the 14-byte region 0xA2-0xAF
//
// 12 + 4 + 4 + 4 + 4 + 101 + 33 + 14 = 176.
type ReceivedGiftEntry struct {
	Reserved [12]byte
	ItemId   int32
	Data1    int32
	Data2    int32
	GiftType int32
	Text     string
	Sender   string
	ItemName string
}

func (m ReceivedGiftEntry) EncodeBytes(l logrus.FieldLogger) []byte {
	w := response.NewWriter(l)
	w.WriteByteArray(m.Reserved[:])
	w.WriteInt32(m.ItemId)
	w.WriteInt32(m.Data1)
	w.WriteInt32(m.Data2)
	w.WriteInt32(m.GiftType)
	model.WritePaddedString(w, m.Text, 101)
	model.WritePaddedString(w, m.Sender, 33)
	model.WritePaddedString(w, m.ItemName, 14)
	return w.Bytes()
}

func DecodeReceivedGiftEntry(r *request.Reader) ReceivedGiftEntry {
	var e ReceivedGiftEntry
	copy(e.Reserved[:], r.ReadBytes(12))
	e.ItemId = r.ReadInt32()
	e.Data1 = r.ReadInt32()
	e.Data2 = r.ReadInt32()
	e.GiftType = r.ReadInt32()
	e.Text = model.ReadPaddedString(r, 101)
	e.Sender = model.ReadPaddedString(r, 33)
	e.ItemName = model.ReadPaddedString(r, 14)
	return e
}

// LoadReceivedGiftDone - LOAD_RECEIVED_GIFT_SUCCESS arm. Handler reads a leading
// flag byte (Decode1 → v26; when 0 the client shows StringPool notice 626 after
// the loop), then a Decode4 count, then count × DecodeBuffer(0xB0) 176-byte
// ReceivedGiftEntry records, then ACKs with COutPacket(0xF5). Wire: mode +
// flag byte + count:uint32 + count × ReceivedGiftEntry(176).
// packet-audit:fname CCashShop::OnCashItemResult#LOAD_RECEIVED_GIFT_SUCCESS
type LoadReceivedGiftDone struct {
	mode  byte
	flag  byte
	gifts []ReceivedGiftEntry
}

func NewLoadReceivedGiftDone(mode byte, flag byte, gifts []ReceivedGiftEntry) LoadReceivedGiftDone {
	return LoadReceivedGiftDone{mode: mode, flag: flag, gifts: gifts}
}

func (m LoadReceivedGiftDone) Mode() byte                 { return m.mode }
func (m LoadReceivedGiftDone) Flag() byte                 { return m.flag }
func (m LoadReceivedGiftDone) Gifts() []ReceivedGiftEntry { return m.gifts }
func (m LoadReceivedGiftDone) Operation() string          { return CashShopOperationWriter }

func (m LoadReceivedGiftDone) String() string {
	return fmt.Sprintf("cash load-received-gift success mode [%d] flag [%d] gifts [%d]", m.mode, m.flag, len(m.gifts))
}

func (m LoadReceivedGiftDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.flag)
		w.WriteInt(uint32(len(m.gifts)))
		for _, g := range m.gifts {
			w.WriteByteArray(g.EncodeBytes(l))
		}
		return w.Bytes()
	}
}

func (m *LoadReceivedGiftDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.flag = r.ReadByte()
		count := int(r.ReadUint32())
		m.gifts = make([]ReceivedGiftEntry, count)
		for i := 0; i < count; i++ {
			m.gifts[i] = DecodeReceivedGiftEntry(r)
		}
	}
}

// ---- mode 96: limit-goods stock changed (CCashShop__OnCashItemResLimitGoodsStockChanged @ 0x48d4d4) ----

// LimitGoodsStockChanged - LIMIT_GOODS_STOCK_CHANGED arm. Handler reads a result
// byte (Decode1 → v4); ONLY when result is 205 or 206 it reads a Decode4 itemId
// and calls UpdateStock/ChangeLimitGoodsState/ChangePage; then NoticeFailReason(result);
// then if result is 177/178/179 SendTransferFieldPacket. Wire: mode + result byte
// + (result∈{205,206}) itemId:uint32. The result byte is a server-controlled
// PROTOCOL status code that gates the conditional itemId read (wire shape), so it
// is a plain field — NOT config-resolved — matching the family's treatment of
// status codes (GachaponOpenDone.resultCode). Config-resolving a byte that
// controls the wire shape is unsafe (the resolved value must equal 205/206 to
// include itemId).
// packet-audit:fname CCashShop::OnCashItemResult#LIMIT_GOODS_STOCK_CHANGED
type LimitGoodsStockChanged struct {
	mode   byte
	result byte
	itemId uint32
}

func NewLimitGoodsStockChanged(mode byte, result byte, itemId uint32) LimitGoodsStockChanged {
	return LimitGoodsStockChanged{mode: mode, result: result, itemId: itemId}
}

func (m LimitGoodsStockChanged) Mode() byte        { return m.mode }
func (m LimitGoodsStockChanged) Result() byte      { return m.result }
func (m LimitGoodsStockChanged) ItemId() uint32    { return m.itemId }
func (m LimitGoodsStockChanged) Operation() string { return CashShopOperationWriter }

// hasStockItemId reports whether the conditional itemId field is present on the
// wire — the client reads it only for result 205 or 206 (sub_48D4D4 @ 0x48d4d4).
func (m LimitGoodsStockChanged) hasStockItemId() bool {
	return m.result == 205 || m.result == 206
}

func (m LimitGoodsStockChanged) String() string {
	return fmt.Sprintf("cash limit-goods-stock-changed mode [%d] result [%d] itemId [%d]", m.mode, m.result, m.itemId)
}

func (m LimitGoodsStockChanged) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.result)
		if m.hasStockItemId() {
			w.WriteInt(m.itemId)
		}
		return w.Bytes()
	}
}

func (m *LimitGoodsStockChanged) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.result = r.ReadByte()
		if m.hasStockItemId() {
			m.itemId = r.ReadUint32()
		}
	}
}

// ---- mode 146: canned notice, bodyless (CCashShop__OnCashItemResShowNotice1089 @ 0x48e6c9) ----

// ShowNotice1089 - SHOW_NOTICE_1089 arm. BODYLESS — the handler reads nothing
// from the packet; it shows StringPool notice 0x1089 (4233) via CUtilDlg::Notice
// (string content not resolved; stripped DEVM build). Wire: mode byte only.
// packet-audit:fname CCashShop::OnCashItemResult#SHOW_NOTICE_1089
type ShowNotice1089 struct {
	mode byte
}

func NewShowNotice1089(mode byte) ShowNotice1089 {
	return ShowNotice1089{mode: mode}
}

func (m ShowNotice1089) Mode() byte        { return m.mode }
func (m ShowNotice1089) Operation() string { return CashShopOperationWriter }

func (m ShowNotice1089) String() string {
	return fmt.Sprintf("cash show-notice-1089 mode [%d]", m.mode)
}

func (m ShowNotice1089) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ShowNotice1089) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ---- mode 147: transfer-world notice reason (CCashShop__OnCashItemResTransferWorldNoticeReason @ 0x48e6f7) ----

// TransferWorldNoticeReason - TRANSFER_WORLD_NOTICE_REASON arm. Handler reads a
// reason byte (Decode1 → v3), calls NoticeFailReason(reason); then if reason is
// 177 or 178 SendTransferFieldPacket. Wire is unconditional: mode + reason byte.
// The reason is a NoticeFailReason-class UI reason code → the body func resolves
// it from the writer "errors" table (sibling failure-arm pattern); the transfer
// side-effect fires only when the resolved byte is 177/178 (client behavior, not
// a wire read). Struct field is a plain byte matching LoadInventoryFailure.errorCode.
// packet-audit:fname CCashShop::OnCashItemResult#TRANSFER_WORLD_NOTICE_REASON
type TransferWorldNoticeReason struct {
	mode      byte
	errorCode byte
}

func NewTransferWorldNoticeReason(mode byte, errorCode byte) TransferWorldNoticeReason {
	return TransferWorldNoticeReason{mode: mode, errorCode: errorCode}
}

func (m TransferWorldNoticeReason) Mode() byte        { return m.mode }
func (m TransferWorldNoticeReason) ErrorCode() byte   { return m.errorCode }
func (m TransferWorldNoticeReason) Operation() string { return CashShopOperationWriter }

func (m TransferWorldNoticeReason) String() string {
	return fmt.Sprintf("cash transfer-world notice reason mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m TransferWorldNoticeReason) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}

func (m *TransferWorldNoticeReason) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// ---- mode 162: locker refresh (CCashShop__OnCashItemResRefreshLocker @ 0x48c321) ----

// RefreshLocker - REFRESH_LOCKER arm. Handler decodes a single GW_CashItemInfo
// via DecodeBuffer(0x37 = 55 bytes) into a fresh locker slot, then refreshes the
// locker window. Wire: mode + CashInventoryItem (55-byte blob, reused via the
// shared model + decodeCashInventoryItemSkipPadding). No leading byte.
// packet-audit:fname CCashShop::OnCashItemResult#REFRESH_LOCKER
type RefreshLocker struct {
	mode byte
	item CashInventoryItem
}

func NewRefreshLocker(mode byte, item CashInventoryItem) RefreshLocker {
	return RefreshLocker{mode: mode, item: item}
}

func (m RefreshLocker) Mode() byte              { return m.mode }
func (m RefreshLocker) Item() CashInventoryItem { return m.item }
func (m RefreshLocker) Operation() string       { return CashShopOperationWriter }

func (m RefreshLocker) String() string {
	return fmt.Sprintf("cash refresh-locker mode [%d] item [%+v]", m.mode, m.item)
}

func (m RefreshLocker) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByteArray(m.item.EncodeBytes(l))
		return w.Bytes()
	}
}

func (m *RefreshLocker) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.item = decodeCashInventoryItemSkipPadding(r)
	}
}

// ---- mode 164: client no-op, bodyless (nullsub_2 @ 0x48c370) ----

// ClientNoOp - CLIENT_NO_OP arm. The dispatcher routes mode 164 to the shared
// nullsub_2 (0x48c370) — a generic empty function that reads nothing and does
// nothing. Genuine client no-op. Modeled as a discrete bodyless struct per INV-1.
// Wire: mode byte only.
// packet-audit:fname CCashShop::OnCashItemResult#CLIENT_NO_OP
type ClientNoOp struct {
	mode byte
}

func NewClientNoOp(mode byte) ClientNoOp {
	return ClientNoOp{mode: mode}
}

func (m ClientNoOp) Mode() byte        { return m.mode }
func (m ClientNoOp) Operation() string { return CashShopOperationWriter }

func (m ClientNoOp) String() string {
	return fmt.Sprintf("cash client no-op mode [%d]", m.mode)
}

func (m ClientNoOp) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ClientNoOp) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ---- mode 166: canned notice, bodyless (CCashShop__OnCashItemResShowNotice1465 @ 0x48c26e) ----

// ShowNotice1465 - SHOW_NOTICE_1465 arm. BODYLESS — the handler reads nothing
// from the packet; it shows StringPool notice 0x1465 (5221) via CUtilDlg::Notice
// (string content not resolved; stripped DEVM build). Wire: mode byte only.
// packet-audit:fname CCashShop::OnCashItemResult#SHOW_NOTICE_1465
type ShowNotice1465 struct {
	mode byte
}

func NewShowNotice1465(mode byte) ShowNotice1465 {
	return ShowNotice1465{mode: mode}
}

func (m ShowNotice1465) Mode() byte        { return m.mode }
func (m ShowNotice1465) Operation() string { return CashShopOperationWriter }

func (m ShowNotice1465) String() string {
	return fmt.Sprintf("cash show-notice-1465 mode [%d]", m.mode)
}

func (m ShowNotice1465) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ShowNotice1465) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ---- mode 167: locker refresh or notice (CCashShop__OnCashItemResRefreshLockerOrNotice @ 0x48c373) ----

// RefreshLockerOrNotice - REFRESH_LOCKER_OR_NOTICE arm. Handler reads a leading
// flag byte (Decode1 → v12), then decodes a single GW_CashItemInfo via
// DecodeBuffer(0x37 = 55 bytes); it then either refreshes the locker window (when
// the client-local flag *(this+1236) is set) or shows StringPool notice 5219/5216
// (selected by the flag byte). Wire: mode + flag byte + CashInventoryItem (55-byte
// blob). The flag is a binary selector (plain byte, like PurchaseRecordDone.purchased).
// packet-audit:fname CCashShop::OnCashItemResult#REFRESH_LOCKER_OR_NOTICE
type RefreshLockerOrNotice struct {
	mode byte
	flag byte
	item CashInventoryItem
}

func NewRefreshLockerOrNotice(mode byte, flag byte, item CashInventoryItem) RefreshLockerOrNotice {
	return RefreshLockerOrNotice{mode: mode, flag: flag, item: item}
}

func (m RefreshLockerOrNotice) Mode() byte              { return m.mode }
func (m RefreshLockerOrNotice) Flag() byte              { return m.flag }
func (m RefreshLockerOrNotice) Item() CashInventoryItem { return m.item }
func (m RefreshLockerOrNotice) Operation() string       { return CashShopOperationWriter }

func (m RefreshLockerOrNotice) String() string {
	return fmt.Sprintf("cash refresh-locker-or-notice mode [%d] flag [%d] item [%+v]", m.mode, m.flag, m.item)
}

func (m RefreshLockerOrNotice) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.flag)
		w.WriteByteArray(m.item.EncodeBytes(l))
		return w.Bytes()
	}
}

func (m *RefreshLockerOrNotice) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.flag = r.ReadByte()
		m.item = decodeCashInventoryItemSkipPadding(r)
	}
}

// ---- mode 168: canned notice, bodyless (CCashShop__OnCashItemResShowNotice1464 @ 0x48c413) ----

// ShowNotice1464 - SHOW_NOTICE_1464 arm. BODYLESS — the handler reads nothing
// from the packet; it shows StringPool notice 0x1464 (5220) via CUtilDlg::Notice
// (string content not resolved; stripped DEVM build). Wire: mode byte only.
// packet-audit:fname CCashShop::OnCashItemResult#SHOW_NOTICE_1464
type ShowNotice1464 struct {
	mode byte
}

func NewShowNotice1464(mode byte) ShowNotice1464 {
	return ShowNotice1464{mode: mode}
}

func (m ShowNotice1464) Mode() byte        { return m.mode }
func (m ShowNotice1464) Operation() string { return CashShopOperationWriter }

func (m ShowNotice1464) String() string {
	return fmt.Sprintf("cash show-notice-1464 mode [%d]", m.mode)
}

func (m ShowNotice1464) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ShowNotice1464) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
