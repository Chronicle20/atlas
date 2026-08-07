package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// Scalar/notice arm family (task-183 Wave 1.4). Every arm in this file is a
// DISCRETE struct even where wire shapes coincide (INV-1) — see
// docs/tasks/task-183-cashshop-result-family/arm-catalog.md for the per-arm
// `fields (v95, decompile-cited)` cell that is the wire-truth for each shape
// below. FREE_CASH_ITEM_DONE is labeled `scalar` in the catalog's coarse
// shape column but the decompile proves a 55-byte GW_CashItemInfo item-blob
// (task-0.3d report) — modeled here as a CashInventoryItem field, not a
// small scalar.

// LimitGoodsCountChanged - LIMIT_GOODS_COUNT_CHANGED arm
// (CCashShop::OnCashItemResLimitGoodsCountChanged): mode + itemId:Decode4
// (int32) + sn:Decode4 (int32) + remainCount:Decode4 (int32). Updates the
// client's local CS_LIMITGOODS array remaining-count.
// packet-audit:fname CCashShop::OnCashItemResult#LIMIT_GOODS_COUNT_CHANGED
type LimitGoodsCountChanged struct {
	mode        byte
	itemId      int32
	sn          int32
	remainCount int32
}

func NewLimitGoodsCountChanged(mode byte, itemId int32, sn int32, remainCount int32) LimitGoodsCountChanged {
	return LimitGoodsCountChanged{mode: mode, itemId: itemId, sn: sn, remainCount: remainCount}
}

func (m LimitGoodsCountChanged) Mode() byte         { return m.mode }
func (m LimitGoodsCountChanged) ItemId() int32      { return m.itemId }
func (m LimitGoodsCountChanged) SN() int32          { return m.sn }
func (m LimitGoodsCountChanged) RemainCount() int32 { return m.remainCount }
func (m LimitGoodsCountChanged) Operation() string  { return CashShopOperationWriter }

func (m LimitGoodsCountChanged) String() string {
	return fmt.Sprintf("cash limit-goods-count-changed mode [%d] itemId [%d] sn [%d] remainCount [%d]", m.mode, m.itemId, m.sn, m.remainCount)
}

func (m LimitGoodsCountChanged) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt32(m.itemId)
		w.WriteInt32(m.sn)
		w.WriteInt32(m.remainCount)
		return w.Bytes()
	}
}

func (m *LimitGoodsCountChanged) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itemId = r.ReadInt32()
		m.sn = r.ReadInt32()
		m.remainCount = r.ReadInt32()
	}
}

// DestroyDone - DESTROY_SUCCESS arm (CCashShop::OnCashItemResDestroyDone):
// mode + sn:DecodeBuffer(8) (int64 LARGE_INTEGER cash-item SN, matched
// against m_aCashItemInfo[i].liSN to find-and-remove the locker entry).
// packet-audit:fname CCashShop::OnCashItemResult#DESTROY_SUCCESS
type DestroyDone struct {
	mode byte
	sn   int64
}

func NewDestroyDone(mode byte, sn int64) DestroyDone {
	return DestroyDone{mode: mode, sn: sn}
}

func (m DestroyDone) Mode() byte        { return m.mode }
func (m DestroyDone) SN() int64         { return m.sn }
func (m DestroyDone) Operation() string { return CashShopOperationWriter }

func (m DestroyDone) String() string {
	return fmt.Sprintf("cash destroy success mode [%d] sn [%d]", m.mode, m.sn)
}

func (m DestroyDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt64(m.sn)
		return w.Bytes()
	}
}

func (m *DestroyDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.sn = r.ReadInt64()
	}
}

// ExpireDone - EXPIRE_DONE arm (CCashShop::OnCashItemResExpireDone): mode +
// sn:DecodeBuffer(8) (int64 LARGE_INTEGER cash-item SN, same shape as
// DestroyDone, matched against m_aCashItemInfo[i].liSN before removal).
// packet-audit:fname CCashShop::OnCashItemResult#EXPIRE_DONE
type ExpireDone struct {
	mode byte
	sn   int64
}

func NewExpireDone(mode byte, sn int64) ExpireDone {
	return ExpireDone{mode: mode, sn: sn}
}

func (m ExpireDone) Mode() byte        { return m.mode }
func (m ExpireDone) SN() int64         { return m.sn }
func (m ExpireDone) Operation() string { return CashShopOperationWriter }

func (m ExpireDone) String() string {
	return fmt.Sprintf("cash expire done mode [%d] sn [%d]", m.mode, m.sn)
}

func (m ExpireDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt64(m.sn)
		return w.Bytes()
	}
}

func (m *ExpireDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.sn = r.ReadInt64()
	}
}

// PurchaseRecordDone - PURCHASE_RECORD arm (CCashShop::OnCashItemResPurchaseRecord):
// mode + goodsSN:Decode4 (int32, used as a ZMap<long,...> key when nonzero) +
// purchased:Decode1 (byte, compared !=0 -> bool, recorded in m_mPurchaseRecord).
// packet-audit:fname CCashShop::OnCashItemResult#PURCHASE_RECORD
type PurchaseRecordDone struct {
	mode      byte
	goodsSN   int32
	purchased byte
}

func NewPurchaseRecordDone(mode byte, goodsSN int32, purchased byte) PurchaseRecordDone {
	return PurchaseRecordDone{mode: mode, goodsSN: goodsSN, purchased: purchased}
}

func (m PurchaseRecordDone) Mode() byte        { return m.mode }
func (m PurchaseRecordDone) GoodsSN() int32    { return m.goodsSN }
func (m PurchaseRecordDone) Purchased() byte   { return m.purchased }
func (m PurchaseRecordDone) Operation() string { return CashShopOperationWriter }

func (m PurchaseRecordDone) String() string {
	return fmt.Sprintf("cash purchase-record success mode [%d] goodsSN [%d] purchased [%d]", m.mode, m.goodsSN, m.purchased)
}

func (m PurchaseRecordDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt32(m.goodsSN)
		w.WriteByte(m.purchased)
		return w.Bytes()
	}
}

func (m *PurchaseRecordDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.goodsSN = r.ReadInt32()
		m.purchased = r.ReadByte()
	}
}

// FreeCashItemDone - FREE_CASH_ITEM_DONE arm (CCashShop::OnCashItemResFreeCashItemDone):
// mode + cashItemInfo:DecodeBuffer(0x37=55) (GW_CashItemInfo blob, decoded
// into a freshly-inserted m_aCashItemInfo slot). The catalog's coarse
// "scalar" shape label is wrong for this arm — the decompile proves a
// 55-byte item-blob (task-0.3d report), reused here as CashInventoryItem.
// packet-audit:fname CCashShop::OnCashItemResult#FREE_CASH_ITEM_DONE
type FreeCashItemDone struct {
	mode byte
	item CashInventoryItem
}

func NewFreeCashItemDone(mode byte, item CashInventoryItem) FreeCashItemDone {
	return FreeCashItemDone{mode: mode, item: item}
}

func (m FreeCashItemDone) Mode() byte              { return m.mode }
func (m FreeCashItemDone) Item() CashInventoryItem { return m.item }
func (m FreeCashItemDone) Operation() string       { return CashShopOperationWriter }

func (m FreeCashItemDone) String() string {
	return fmt.Sprintf("cash free-cash-item done mode [%d] item [%+v]", m.mode, m.item)
}

func (m FreeCashItemDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByteArray(m.item.EncodeBytes(l))
		return w.Bytes()
	}
}

func (m *FreeCashItemDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.item = decodeCashInventoryItemSkipPadding(r)
	}
}
