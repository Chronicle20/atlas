package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// Gachapon arm family (task-183 Wave 1.4). Both arms carry a CONDITIONAL
// single GW_CashItemInfo (55-byte) blob, gated on a leading flag byte read
// earlier in the same handler — the flag(s) are modeled as ordinary struct
// fields, and Encode writes the blob iff the flag condition holds (mirroring
// the client's own conditional read). Present only in v84/v87/v95 among
// MODERN versions (n-a v83, n-a jms per arm-catalog.md — verifiably absent
// from the whole binary in both). See
// docs/tasks/task-183-cashshop-result-family/arm-catalog.md and
// task-0.3e-report.md for the wire-truth read order.

// GachaponOpenDone - GACHAPON_OPEN_SUCCESS arm
// (CCashShop::OnCashItemResCashGachaponOpenDone): mode +
// sn:DecodeBuffer(8) (int64 LARGE_INTEGER, matched against existing
// m_aCashItemInfo[i].liSN) + remain:Decode4 (int32, new qty — 0 removes the
// locker entry) + isCashItem:Decode1 (byte flag) + CONDITIONAL if
// isCashItem!=0: newItem:DecodeBuffer(0x37=55) (single GW_CashItemInfo blob,
// appended to m_aCashItemInfo) + resultCode:Decode4 (int32, passed to
// CUICashGachapon::OnCashGachaponOpenResult) + resultParam2:Decode1 (byte,
// second param to the same call).
// packet-audit:fname CCashShop::OnCashItemResult#GACHAPON_OPEN_SUCCESS
type GachaponOpenDone struct {
	mode         byte
	sn           int64
	remain       int32
	isCashItem   byte
	newItem      CashInventoryItem
	resultCode   int32
	resultParam2 byte
}

func NewGachaponOpenDone(mode byte, sn int64, remain int32, isCashItem byte, newItem CashInventoryItem, resultCode int32, resultParam2 byte) GachaponOpenDone {
	return GachaponOpenDone{mode: mode, sn: sn, remain: remain, isCashItem: isCashItem, newItem: newItem, resultCode: resultCode, resultParam2: resultParam2}
}

func (m GachaponOpenDone) Mode() byte                 { return m.mode }
func (m GachaponOpenDone) SN() int64                  { return m.sn }
func (m GachaponOpenDone) Remain() int32              { return m.remain }
func (m GachaponOpenDone) IsCashItem() byte           { return m.isCashItem }
func (m GachaponOpenDone) NewItem() CashInventoryItem { return m.newItem }
func (m GachaponOpenDone) ResultCode() int32          { return m.resultCode }
func (m GachaponOpenDone) ResultParam2() byte         { return m.resultParam2 }
func (m GachaponOpenDone) Operation() string          { return CashShopOperationWriter }

func (m GachaponOpenDone) String() string {
	return fmt.Sprintf("cash gachapon-open success mode [%d] sn [%d] remain [%d] isCashItem [%d] resultCode [%d] resultParam2 [%d]", m.mode, m.sn, m.remain, m.isCashItem, m.resultCode, m.resultParam2)
}

func (m GachaponOpenDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt64(m.sn)
		w.WriteInt32(m.remain)
		w.WriteByte(m.isCashItem)
		if m.isCashItem != 0 {
			w.WriteByteArray(m.newItem.EncodeBytes(l))
		}
		w.WriteInt32(m.resultCode)
		w.WriteByte(m.resultParam2)
		return w.Bytes()
	}
}

func (m *GachaponOpenDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.sn = r.ReadInt64()
		m.remain = r.ReadInt32()
		m.isCashItem = r.ReadByte()
		if m.isCashItem != 0 {
			m.newItem = decodeCashInventoryItemSkipPadding(r)
		}
		m.resultCode = r.ReadInt32()
		m.resultParam2 = r.ReadByte()
	}
}

// GachaponCopyDone - GACHAPON_COPY_SUCCESS arm
// (CCashShop::OnCashItemResCashGachaponCopyDone): mode + flag1:Decode1
// (byte) + flag2:Decode1 (byte) + unused1:Decode4 (int32, discarded) +
// unused2:Decode4 (int32, discarded) + lostItemId:Decode4 (int32,
// nRandomItemLostItemID) + lostNumber:Decode4 (int32, nRandomItemLostNumber)
// + CONDITIONAL if flag1!=0 AND flag2!=0: item:DecodeBuffer(0x37=55) (single
// GW_CashItemInfo blob, appended to m_aCashItemInfo). Conditional item-blob
// gated on a compound (AND) boolean condition — distinct from
// GachaponOpenDone's single-flag gate.
// packet-audit:fname CCashShop::OnCashItemResult#GACHAPON_COPY_SUCCESS
type GachaponCopyDone struct {
	mode       byte
	flag1      byte
	flag2      byte
	unused1    int32
	unused2    int32
	lostItemId int32
	lostNumber int32
	item       CashInventoryItem
}

func NewGachaponCopyDone(mode byte, flag1 byte, flag2 byte, unused1 int32, unused2 int32, lostItemId int32, lostNumber int32, item CashInventoryItem) GachaponCopyDone {
	return GachaponCopyDone{mode: mode, flag1: flag1, flag2: flag2, unused1: unused1, unused2: unused2, lostItemId: lostItemId, lostNumber: lostNumber, item: item}
}

func (m GachaponCopyDone) Mode() byte              { return m.mode }
func (m GachaponCopyDone) Flag1() byte             { return m.flag1 }
func (m GachaponCopyDone) Flag2() byte             { return m.flag2 }
func (m GachaponCopyDone) Unused1() int32          { return m.unused1 }
func (m GachaponCopyDone) Unused2() int32          { return m.unused2 }
func (m GachaponCopyDone) LostItemId() int32       { return m.lostItemId }
func (m GachaponCopyDone) LostNumber() int32       { return m.lostNumber }
func (m GachaponCopyDone) Item() CashInventoryItem { return m.item }
func (m GachaponCopyDone) Operation() string       { return CashShopOperationWriter }

func (m GachaponCopyDone) String() string {
	return fmt.Sprintf("cash gachapon-copy success mode [%d] flag1 [%d] flag2 [%d] lostItemId [%d] lostNumber [%d]", m.mode, m.flag1, m.flag2, m.lostItemId, m.lostNumber)
}

func (m GachaponCopyDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.flag1)
		w.WriteByte(m.flag2)
		w.WriteInt32(m.unused1)
		w.WriteInt32(m.unused2)
		w.WriteInt32(m.lostItemId)
		w.WriteInt32(m.lostNumber)
		if m.flag1 != 0 && m.flag2 != 0 {
			w.WriteByteArray(m.item.EncodeBytes(l))
		}
		return w.Bytes()
	}
}

func (m *GachaponCopyDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.flag1 = r.ReadByte()
		m.flag2 = r.ReadByte()
		m.unused1 = r.ReadInt32()
		m.unused2 = r.ReadInt32()
		m.lostItemId = r.ReadInt32()
		m.lostNumber = r.ReadInt32()
		if m.flag1 != 0 && m.flag2 != 0 {
			m.item = decodeCashInventoryItemSkipPadding(r)
		}
	}
}
