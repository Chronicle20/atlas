package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Gift/coupon/package item-blob arm family (task-183 Wave 1.3). Every arm in
// this file is a DISCRETE struct even where the wire shape is identical to
// another arm (INV-1) — see docs/tasks/task-183-cashshop-result-family/
// arm-catalog.md for the per-arm `fields (v95, decompile-cited)` cell that is
// the wire-truth for each shape below. The catalog's coarse "item-blob" shape
// label is a reasoning aid only; several of these arms carry NO item-blob at
// all (GIFT_SUCCESS, REBATE_SUCCESS, GIFT_PACKAGE_SUCCESS, BUY_NORMAL_SUCCESS)
// — see task-0.3e/0.3f reports.

// GiftListEntry represents a single GW_GiftList record (98 bytes), the
// per-entry wire shape of LOAD_GIFT_SUCCESS's count-prefixed gift list.
// Confirmed via `type_inspect("GW_GiftList")` in the v95 IDB, cross-checked
// against decompile+disasm (task-0.3f report): liSN offset 0 (8B),
// nItemID offset 8 (4B int32), sBuyCharacterName (a.k.a. sSendCharacterName
// per CUIReceiveGift::SetValues) offset 12 (13B fixed null-terminated),
// sText offset 25 (73B fixed null-terminated). 8+4+13+73=98. NOT the same
// blob as GW_CashItemInfo/CashInventoryItem — do not conflate the two.
type GiftListEntry struct {
	SN               int64
	ItemId           int32
	BuyCharacterName string
	Text             string
}

func (m GiftListEntry) EncodeBytes(l logrus.FieldLogger) []byte {
	w := response.NewWriter(l)
	w.WriteInt64(m.SN)
	w.WriteInt32(m.ItemId)
	model.WritePaddedString(w, m.BuyCharacterName, 13)
	model.WritePaddedString(w, m.Text, 73)
	return w.Bytes()
}

func DecodeGiftListEntry(r *request.Reader) GiftListEntry {
	return GiftListEntry{
		SN:               r.ReadInt64(),
		ItemId:           r.ReadInt32(),
		BuyCharacterName: model.ReadPaddedString(r, 13),
		Text:             model.ReadPaddedString(r, 73),
	}
}

// PackedCashItemRef represents the 8-byte packed `_ULARGE_INTEGER`-shaped
// record used by USE_COUPON_SUCCESS's second list and BUY_NORMAL_SUCCESS's
// list: quantity offset 0 (u16), slotPos offset 2 (u16), itemId offset 4
// (i32, full 32-bit — itemId/1000000 derives an inventory-tab category
// client-side). Bit layout resolved from raw disassembly, NOT from the
// (unreliable, mis-scaled) Hex-Rays pseudocode pointer arithmetic — see
// task-0.3f report arms 2/3. NOT a plain 8-byte SN.
type PackedCashItemRef struct {
	Quantity uint16
	SlotPos  uint16
	ItemId   int32
}

func (m PackedCashItemRef) EncodeBytes(l logrus.FieldLogger) []byte {
	w := response.NewWriter(l)
	w.WriteShort(m.Quantity)
	w.WriteShort(m.SlotPos)
	w.WriteInt32(m.ItemId)
	return w.Bytes()
}

func DecodePackedCashItemRef(r *request.Reader) PackedCashItemRef {
	return PackedCashItemRef{
		Quantity: r.ReadUint16(),
		SlotPos:  r.ReadUint16(),
		ItemId:   r.ReadInt32(),
	}
}

// giftHasNxCashSpent reports whether this arm's trailing nxCashSpent:Decode4
// field is present on the wire. Present on every audited GMS version
// (v83/v84/v87/v95); jms's resolved handler stops after quantity/unused2 and
// never reads it (task-183 arm-catalog.md "Per-version wire divergences" §2,
// task-0.4-jms-modes.md per-arm table — GIFT_SUCCESS/GIFT_PACKAGE_SUCCESS
// rows). Region-gated, not version-gated, per that finding.
func giftHasNxCashSpent(t tenant.Model) bool {
	return t.Region() == "GMS"
}

// GiftDone - GIFT_SUCCESS arm (CCashShop::OnCashItemResGiftDone): mode +
// recipientName:DecodeStr + itemId:Decode4 (int32, name-lookup key only —
// the item is never inserted into m_aCashItemInfo) + quantity:Decode2
// (uint16) + nxCashSpent:Decode4 (int32, GMS only — see giftHasNxCashSpent).
// TRUE SHAPE per task-0.3e report: NO item-blob at all — pure scalar body.
// This resolves the legacy 0x4D gift TODO (the old CashShopGifts stub
// modeled the wrong shape entirely). jms's resolved handler lacks the
// trailing nxCashSpent field (task-183 arm-catalog.md "Per-version wire
// divergences" §2) — gated Task 1.5.
// packet-audit:fname CCashShop::OnCashItemResult#GIFT_SUCCESS
type GiftDone struct {
	mode          byte
	recipientName string
	itemId        int32
	quantity      uint16
	nxCashSpent   int32
}

func NewGiftDone(mode byte, recipientName string, itemId int32, quantity uint16, nxCashSpent int32) GiftDone {
	return GiftDone{mode: mode, recipientName: recipientName, itemId: itemId, quantity: quantity, nxCashSpent: nxCashSpent}
}

func (m GiftDone) Mode() byte            { return m.mode }
func (m GiftDone) RecipientName() string { return m.recipientName }
func (m GiftDone) ItemId() int32         { return m.itemId }
func (m GiftDone) Quantity() uint16      { return m.quantity }
func (m GiftDone) NxCashSpent() int32    { return m.nxCashSpent }
func (m GiftDone) Operation() string     { return CashShopOperationWriter }

func (m GiftDone) String() string {
	return fmt.Sprintf("cash gift success mode [%d] recipientName [%s] itemId [%d] quantity [%d] nxCashSpent [%d]", m.mode, m.recipientName, m.itemId, m.quantity, m.nxCashSpent)
}

func (m GiftDone) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.recipientName)
		w.WriteInt32(m.itemId)
		w.WriteShort(m.quantity)
		if giftHasNxCashSpent(t) {
			w.WriteInt32(m.nxCashSpent)
		}
		return w.Bytes()
	}
}

func (m *GiftDone) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.recipientName = r.ReadAsciiString()
		m.itemId = r.ReadInt32()
		m.quantity = r.ReadUint16()
		if giftHasNxCashSpent(t) {
			m.nxCashSpent = r.ReadInt32()
		}
	}
}

// LoadGiftDone - LOAD_GIFT_SUCCESS arm (CCashShop::OnCashItemResLoadGiftDone):
// mode + count:Decode2 (uint16) + count-prefixed LIST of GiftListEntry
// (98-byte GW_GiftList records). See task-0.3e/0.3f reports.
// packet-audit:fname CCashShop::OnCashItemResult#LOAD_GIFT_SUCCESS
type LoadGiftDone struct {
	mode  byte
	gifts []GiftListEntry
}

func NewLoadGiftDone(mode byte, gifts []GiftListEntry) LoadGiftDone {
	return LoadGiftDone{mode: mode, gifts: gifts}
}

func (m LoadGiftDone) Mode() byte             { return m.mode }
func (m LoadGiftDone) Gifts() []GiftListEntry { return m.gifts }
func (m LoadGiftDone) Operation() string      { return CashShopOperationWriter }

func (m LoadGiftDone) String() string {
	return fmt.Sprintf("cash load-gift success mode [%d] gifts [%d]", m.mode, len(m.gifts))
}

func (m LoadGiftDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteShort(uint16(len(m.gifts)))
		for _, g := range m.gifts {
			w.WriteByteArray(g.EncodeBytes(l))
		}
		return w.Bytes()
	}
}

func (m *LoadGiftDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := int(r.ReadUint16())
		m.gifts = make([]GiftListEntry, count)
		for i := 0; i < count; i++ {
			m.gifts[i] = DecodeGiftListEntry(r)
		}
	}
}

// UseCouponDone - USE_COUPON_SUCCESS arm (CCashShop::OnCashItemResUseCouponDone):
// mode + itemCount:Decode1 (byte) + count-prefixed LIST of CashInventoryItem
// (55-byte GW_CashItemInfo) + maplePoint:Decode4 (int32) + uliCount:Decode4
// (int32) + count-prefixed LIST of PackedCashItemRef (8-byte packed record) +
// meso:Decode4 (int32). See task-0.3e/0.3f reports.
// packet-audit:fname CCashShop::OnCashItemResult#USE_COUPON_SUCCESS
type UseCouponDone struct {
	mode       byte
	items      []CashInventoryItem
	maplePoint int32
	refs       []PackedCashItemRef
	meso       int32
}

func NewUseCouponDone(mode byte, items []CashInventoryItem, maplePoint int32, refs []PackedCashItemRef, meso int32) UseCouponDone {
	return UseCouponDone{mode: mode, items: items, maplePoint: maplePoint, refs: refs, meso: meso}
}

func (m UseCouponDone) Mode() byte                 { return m.mode }
func (m UseCouponDone) Items() []CashInventoryItem { return m.items }
func (m UseCouponDone) MaplePoint() int32          { return m.maplePoint }
func (m UseCouponDone) Refs() []PackedCashItemRef  { return m.refs }
func (m UseCouponDone) Meso() int32                { return m.meso }
func (m UseCouponDone) Operation() string          { return CashShopOperationWriter }

func (m UseCouponDone) String() string {
	return fmt.Sprintf("cash use-coupon success mode [%d] items [%d] maplePoint [%d] refs [%d] meso [%d]", m.mode, len(m.items), m.maplePoint, len(m.refs), m.meso)
}

func (m UseCouponDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(byte(len(m.items)))
		for _, item := range m.items {
			w.WriteByteArray(item.EncodeBytes(l))
		}
		w.WriteInt32(m.maplePoint)
		w.WriteInt32(int32(len(m.refs)))
		for _, ref := range m.refs {
			w.WriteByteArray(ref.EncodeBytes(l))
		}
		w.WriteInt32(m.meso)
		return w.Bytes()
	}
}

func (m *UseCouponDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		itemCount := int(r.ReadByte())
		m.items = make([]CashInventoryItem, itemCount)
		for i := 0; i < itemCount; i++ {
			m.items[i] = decodeCashInventoryItemSkipPadding(r)
		}
		m.maplePoint = r.ReadInt32()
		uliCount := int(r.ReadInt32())
		m.refs = make([]PackedCashItemRef, uliCount)
		for i := 0; i < uliCount; i++ {
			m.refs[i] = DecodePackedCashItemRef(r)
		}
		m.meso = r.ReadInt32()
	}
}

// GiftCouponDone - GIFT_COUPON_SUCCESS arm (CCashShop::OnCashItemResGiftCouponDone):
// mode + recipientName:DecodeStr + itemCount:Decode1 (byte) + count-prefixed
// LIST of CashInventoryItem (55-byte) + maplePoint:Decode4 (int32).
// packet-audit:fname CCashShop::OnCashItemResult#GIFT_COUPON_SUCCESS
type GiftCouponDone struct {
	mode          byte
	recipientName string
	items         []CashInventoryItem
	maplePoint    int32
}

func NewGiftCouponDone(mode byte, recipientName string, items []CashInventoryItem, maplePoint int32) GiftCouponDone {
	return GiftCouponDone{mode: mode, recipientName: recipientName, items: items, maplePoint: maplePoint}
}

func (m GiftCouponDone) Mode() byte                 { return m.mode }
func (m GiftCouponDone) RecipientName() string      { return m.recipientName }
func (m GiftCouponDone) Items() []CashInventoryItem { return m.items }
func (m GiftCouponDone) MaplePoint() int32          { return m.maplePoint }
func (m GiftCouponDone) Operation() string          { return CashShopOperationWriter }

func (m GiftCouponDone) String() string {
	return fmt.Sprintf("cash gift-coupon success mode [%d] recipientName [%s] items [%d] maplePoint [%d]", m.mode, m.recipientName, len(m.items), m.maplePoint)
}

func (m GiftCouponDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.recipientName)
		w.WriteByte(byte(len(m.items)))
		for _, item := range m.items {
			w.WriteByteArray(item.EncodeBytes(l))
		}
		w.WriteInt32(m.maplePoint)
		return w.Bytes()
	}
}

func (m *GiftCouponDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.recipientName = r.ReadAsciiString()
		itemCount := int(r.ReadByte())
		m.items = make([]CashInventoryItem, itemCount)
		for i := 0; i < itemCount; i++ {
			m.items[i] = decodeCashInventoryItemSkipPadding(r)
		}
		m.maplePoint = r.ReadInt32()
	}
}

// BuyPackageDone - BUY_PACKAGE_SUCCESS arm (CCashShop::OnCashItemResBuyPackageDone):
// mode + itemCount:Decode1 (byte) + count-prefixed LIST of CashInventoryItem
// (55-byte) + trailingCount:Decode2 (uint16, branches notice-text format).
// packet-audit:fname CCashShop::OnCashItemResult#BUY_PACKAGE_SUCCESS
type BuyPackageDone struct {
	mode          byte
	items         []CashInventoryItem
	trailingCount uint16
}

func NewBuyPackageDone(mode byte, items []CashInventoryItem, trailingCount uint16) BuyPackageDone {
	return BuyPackageDone{mode: mode, items: items, trailingCount: trailingCount}
}

func (m BuyPackageDone) Mode() byte                 { return m.mode }
func (m BuyPackageDone) Items() []CashInventoryItem { return m.items }
func (m BuyPackageDone) TrailingCount() uint16      { return m.trailingCount }
func (m BuyPackageDone) Operation() string          { return CashShopOperationWriter }

func (m BuyPackageDone) String() string {
	return fmt.Sprintf("cash buy-package success mode [%d] items [%d] trailingCount [%d]", m.mode, len(m.items), m.trailingCount)
}

func (m BuyPackageDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(byte(len(m.items)))
		for _, item := range m.items {
			w.WriteByteArray(item.EncodeBytes(l))
		}
		w.WriteShort(m.trailingCount)
		return w.Bytes()
	}
}

func (m *BuyPackageDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		itemCount := int(r.ReadByte())
		m.items = make([]CashInventoryItem, itemCount)
		for i := 0; i < itemCount; i++ {
			m.items[i] = decodeCashInventoryItemSkipPadding(r)
		}
		m.trailingCount = r.ReadUint16()
	}
}

// GiftPackageDone - GIFT_PACKAGE_SUCCESS arm (CCashShop::OnCashItemResGiftPackageDone):
// mode + recipientName:DecodeStr + packageId:Decode4 (int32, CItemInfo::GetSpecialName
// key) + unused1:Decode2 (uint16, read+discarded client-side but present on
// the wire) + unused2:Decode2 (uint16, same) + nxCashSpent:Decode4 (int32,
// GMS only — see giftHasNxCashSpent). TRUE SHAPE per task-0.3e report: NO
// item-blob (contra the catalog's coarse shape label). jms lacks the
// trailing nxCashSpent field (arm-catalog.md divergence §2) — gated Task 1.5.
// packet-audit:fname CCashShop::OnCashItemResult#GIFT_PACKAGE_SUCCESS
type GiftPackageDone struct {
	mode          byte
	recipientName string
	packageId     int32
	unused1       uint16
	unused2       uint16
	nxCashSpent   int32
}

func NewGiftPackageDone(mode byte, recipientName string, packageId int32, unused1 uint16, unused2 uint16, nxCashSpent int32) GiftPackageDone {
	return GiftPackageDone{mode: mode, recipientName: recipientName, packageId: packageId, unused1: unused1, unused2: unused2, nxCashSpent: nxCashSpent}
}

func (m GiftPackageDone) Mode() byte            { return m.mode }
func (m GiftPackageDone) RecipientName() string { return m.recipientName }
func (m GiftPackageDone) PackageId() int32      { return m.packageId }
func (m GiftPackageDone) Unused1() uint16       { return m.unused1 }
func (m GiftPackageDone) Unused2() uint16       { return m.unused2 }
func (m GiftPackageDone) NxCashSpent() int32    { return m.nxCashSpent }
func (m GiftPackageDone) Operation() string     { return CashShopOperationWriter }

func (m GiftPackageDone) String() string {
	return fmt.Sprintf("cash gift-package success mode [%d] recipientName [%s] packageId [%d] nxCashSpent [%d]", m.mode, m.recipientName, m.packageId, m.nxCashSpent)
}

func (m GiftPackageDone) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.recipientName)
		w.WriteInt32(m.packageId)
		w.WriteShort(m.unused1)
		w.WriteShort(m.unused2)
		if giftHasNxCashSpent(t) {
			w.WriteInt32(m.nxCashSpent)
		}
		return w.Bytes()
	}
}

func (m *GiftPackageDone) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.recipientName = r.ReadAsciiString()
		m.packageId = r.ReadInt32()
		m.unused1 = r.ReadUint16()
		m.unused2 = r.ReadUint16()
		if giftHasNxCashSpent(t) {
			m.nxCashSpent = r.ReadInt32()
		}
	}
}

// BuyNormalDone - BUY_NORMAL_SUCCESS arm (CCashShop::OnCashItemResBuyNormalDone):
// mode + count:Decode4 (int32) + count-prefixed LIST of PackedCashItemRef
// (8-byte packed record, SAME shape as USE_COUPON_SUCCESS's second list).
// TRUE SHAPE per task-0.3e/0.3f reports: NO GW_CashItemInfo item-blob (contra
// the catalog's coarse shape label); this handler reads the count as int32
// (not byte, unlike the other list arms in this file).
// packet-audit:fname CCashShop::OnCashItemResult#BUY_NORMAL_SUCCESS
type BuyNormalDone struct {
	mode byte
	refs []PackedCashItemRef
}

func NewBuyNormalDone(mode byte, refs []PackedCashItemRef) BuyNormalDone {
	return BuyNormalDone{mode: mode, refs: refs}
}

func (m BuyNormalDone) Mode() byte                { return m.mode }
func (m BuyNormalDone) Refs() []PackedCashItemRef { return m.refs }
func (m BuyNormalDone) Operation() string         { return CashShopOperationWriter }

func (m BuyNormalDone) String() string {
	return fmt.Sprintf("cash buy-normal success mode [%d] refs [%d]", m.mode, len(m.refs))
}

func (m BuyNormalDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt32(int32(len(m.refs)))
		for _, ref := range m.refs {
			w.WriteByteArray(ref.EncodeBytes(l))
		}
		return w.Bytes()
	}
}

func (m *BuyNormalDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := int(r.ReadInt32())
		m.refs = make([]PackedCashItemRef, count)
		for i := 0; i < count; i++ {
			m.refs[i] = DecodePackedCashItemRef(r)
		}
	}
}

// FriendshipDone - FRIENDSHIP_SUCCESS arm (CCashShop::OnCashItemResFriendShipDone):
// mode + item:DecodeBuffer(55) (single CashInventoryItem blob, appended to
// m_aCashItemInfo) + recipientName:DecodeStr + itemId:Decode4 (int32) +
// quantity:Decode2 (uint16). Byte-for-byte identical shape to CoupleDone —
// modeled as a separate discrete struct per INV-1.
// packet-audit:fname CCashShop::OnCashItemResult#FRIENDSHIP_SUCCESS
type FriendshipDone struct {
	mode          byte
	item          CashInventoryItem
	recipientName string
	itemId        int32
	quantity      uint16
}

func NewFriendshipDone(mode byte, item CashInventoryItem, recipientName string, itemId int32, quantity uint16) FriendshipDone {
	return FriendshipDone{mode: mode, item: item, recipientName: recipientName, itemId: itemId, quantity: quantity}
}

func (m FriendshipDone) Mode() byte              { return m.mode }
func (m FriendshipDone) Item() CashInventoryItem { return m.item }
func (m FriendshipDone) RecipientName() string   { return m.recipientName }
func (m FriendshipDone) ItemId() int32           { return m.itemId }
func (m FriendshipDone) Quantity() uint16        { return m.quantity }
func (m FriendshipDone) Operation() string       { return CashShopOperationWriter }

func (m FriendshipDone) String() string {
	return fmt.Sprintf("cash friendship success mode [%d] recipientName [%s] itemId [%d] quantity [%d]", m.mode, m.recipientName, m.itemId, m.quantity)
}

func (m FriendshipDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByteArray(m.item.EncodeBytes(l))
		w.WriteAsciiString(m.recipientName)
		w.WriteInt32(m.itemId)
		w.WriteShort(m.quantity)
		return w.Bytes()
	}
}

func (m *FriendshipDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.item = decodeCashInventoryItemSkipPadding(r)
		m.recipientName = r.ReadAsciiString()
		m.itemId = r.ReadInt32()
		m.quantity = r.ReadUint16()
	}
}

// RebateDone - REBATE_SUCCESS arm (CCashShop::OnCashItemResRebateDone): mode +
// sn:DecodeBuffer(8) (int64 LARGE_INTEGER, matched against existing
// m_aCashItemInfo[i].liSN client-side to find-and-remove the locker entry —
// same pattern as DESTROY_SUCCESS/EXPIRE_DONE) + amount:Decode4 (int32).
// TRUE SHAPE per task-0.3e report: NO item-blob (contra the catalog's coarse
// shape label).
// packet-audit:fname CCashShop::OnCashItemResult#REBATE_SUCCESS
type RebateDone struct {
	mode   byte
	sn     int64
	amount int32
}

func NewRebateDone(mode byte, sn int64, amount int32) RebateDone {
	return RebateDone{mode: mode, sn: sn, amount: amount}
}

func (m RebateDone) Mode() byte        { return m.mode }
func (m RebateDone) SN() int64         { return m.sn }
func (m RebateDone) Amount() int32     { return m.amount }
func (m RebateDone) Operation() string { return CashShopOperationWriter }

func (m RebateDone) String() string {
	return fmt.Sprintf("cash rebate success mode [%d] sn [%d] amount [%d]", m.mode, m.sn, m.amount)
}

func (m RebateDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt64(m.sn)
		w.WriteInt32(m.amount)
		return w.Bytes()
	}
}

func (m *RebateDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.sn = r.ReadInt64()
		m.amount = r.ReadInt32()
	}
}

// CoupleDone - COUPLE_SUCCESS arm (CCashShop::OnCashItemResCoupleDone): mode +
// item:DecodeBuffer(55) (single CashInventoryItem blob, appended to
// m_aCashItemInfo) + recipientName:DecodeStr + itemId:Decode4 (int32,
// second/separate item reference used only for CItemInfo::GetItemName in the
// notice text) + quantity:Decode2 (uint16). Genuine item-blob arm — single
// blob plus trailing gift-notice scalars.
// packet-audit:fname CCashShop::OnCashItemResult#COUPLE_SUCCESS
type CoupleDone struct {
	mode          byte
	item          CashInventoryItem
	recipientName string
	itemId        int32
	quantity      uint16
}

func NewCoupleDone(mode byte, item CashInventoryItem, recipientName string, itemId int32, quantity uint16) CoupleDone {
	return CoupleDone{mode: mode, item: item, recipientName: recipientName, itemId: itemId, quantity: quantity}
}

func (m CoupleDone) Mode() byte              { return m.mode }
func (m CoupleDone) Item() CashInventoryItem { return m.item }
func (m CoupleDone) RecipientName() string   { return m.recipientName }
func (m CoupleDone) ItemId() int32           { return m.itemId }
func (m CoupleDone) Quantity() uint16        { return m.quantity }
func (m CoupleDone) Operation() string       { return CashShopOperationWriter }

func (m CoupleDone) String() string {
	return fmt.Sprintf("cash couple success mode [%d] recipientName [%s] itemId [%d] quantity [%d]", m.mode, m.recipientName, m.itemId, m.quantity)
}

func (m CoupleDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByteArray(m.item.EncodeBytes(l))
		w.WriteAsciiString(m.recipientName)
		w.WriteInt32(m.itemId)
		w.WriteShort(m.quantity)
		return w.Bytes()
	}
}

func (m *CoupleDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.item = decodeCashInventoryItemSkipPadding(r)
		m.recipientName = r.ReadAsciiString()
		m.itemId = r.ReadInt32()
		m.quantity = r.ReadUint16()
	}
}
