package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// Transfer/name-change/maple-point arm family (task-183 Wave 1.4). Every arm
// in this file is a DISCRETE struct even where wire shapes coincide (INV-1)
// — see docs/tasks/task-183-cashshop-result-family/arm-catalog.md for the
// per-arm `fields (v95, decompile-cited)` cell that is the wire-truth for
// each shape below. NAME_CHANGE_BUY_DONE and TRANSFER_WORLD_SUCCESS are
// labeled `scalar` in the catalog's coarse shape column but the decompile
// proves a 55-byte GW_CashItemInfo item-blob (task-0.3d report) — modeled
// here as a CashInventoryItem field, not a small scalar.

// NameChangeBuyDone - NAME_CHANGE_BUY_DONE arm
// (CCashShop::OnCashItemNameChangeResBuyDone): mode +
// cashItemInfo:DecodeBuffer(0x37=55) (GW_CashItemInfo blob, decoded into a
// freshly-inserted m_aCashItemInfo slot). Body then branches on
// m_dwAvatarPurchaseOption (client-local state, no packet read).
// packet-audit:fname CCashShop::OnCashItemResult#NAME_CHANGE_BUY_DONE
type NameChangeBuyDone struct {
	mode byte
	item CashInventoryItem
}

func NewNameChangeBuyDone(mode byte, item CashInventoryItem) NameChangeBuyDone {
	return NameChangeBuyDone{mode: mode, item: item}
}

func (m NameChangeBuyDone) Mode() byte              { return m.mode }
func (m NameChangeBuyDone) Item() CashInventoryItem { return m.item }
func (m NameChangeBuyDone) Operation() string       { return CashShopOperationWriter }

func (m NameChangeBuyDone) String() string {
	return fmt.Sprintf("cash name-change buy-done mode [%d] item [%+v]", m.mode, m.item)
}

func (m NameChangeBuyDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByteArray(m.item.EncodeBytes(l))
		return w.Bytes()
	}
}

func (m *NameChangeBuyDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.item = decodeCashInventoryItemSkipPadding(r)
	}
}

// TransferWorldDone - TRANSFER_WORLD_SUCCESS arm
// (CCashShop::OnCashItemResTransferWorldDone): mode +
// cashItemInfo:DecodeBuffer(0x37=55) (GW_CashItemInfo blob). Byte-identical
// body shape to NameChangeBuyDone (same blob, same m_dwAvatarPurchaseOption
// branch, different singleton follow-up) — modeled as a separate discrete
// struct per INV-1.
// packet-audit:fname CCashShop::OnCashItemResult#TRANSFER_WORLD_SUCCESS
type TransferWorldDone struct {
	mode byte
	item CashInventoryItem
}

func NewTransferWorldDone(mode byte, item CashInventoryItem) TransferWorldDone {
	return TransferWorldDone{mode: mode, item: item}
}

func (m TransferWorldDone) Mode() byte              { return m.mode }
func (m TransferWorldDone) Item() CashInventoryItem { return m.item }
func (m TransferWorldDone) Operation() string       { return CashShopOperationWriter }

func (m TransferWorldDone) String() string {
	return fmt.Sprintf("cash transfer-world success mode [%d] item [%+v]", m.mode, m.item)
}

func (m TransferWorldDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByteArray(m.item.EncodeBytes(l))
		return w.Bytes()
	}
}

func (m *TransferWorldDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.item = decodeCashInventoryItemSkipPadding(r)
	}
}

// ChangeMaplePointDone - CHANGE_MAPLE_POINT_SUCCESS arm
// (CCashShop::OnCashItemResChangeMaplePointDone): mode +
// sn:DecodeBuffer(8) (int64 LARGE_INTEGER cash-item SN, matched against
// m_aCashItemInfo[i].liSN to decrement/remove the entry's nNumber) +
// count:Decode4 (int32, formatted directly into the notice string).
// Present only in v84/v87/v95 among MODERN versions (n-a v83, n-a jms per
// arm-catalog.md).
// packet-audit:fname CCashShop::OnCashItemResult#CHANGE_MAPLE_POINT_SUCCESS
type ChangeMaplePointDone struct {
	mode  byte
	sn    int64
	count int32
}

func NewChangeMaplePointDone(mode byte, sn int64, count int32) ChangeMaplePointDone {
	return ChangeMaplePointDone{mode: mode, sn: sn, count: count}
}

func (m ChangeMaplePointDone) Mode() byte        { return m.mode }
func (m ChangeMaplePointDone) SN() int64         { return m.sn }
func (m ChangeMaplePointDone) Count() int32      { return m.count }
func (m ChangeMaplePointDone) Operation() string { return CashShopOperationWriter }

func (m ChangeMaplePointDone) String() string {
	return fmt.Sprintf("cash change-maple-point success mode [%d] sn [%d] count [%d]", m.mode, m.sn, m.count)
}

func (m ChangeMaplePointDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt64(m.sn)
		w.WriteInt32(m.count)
		return w.Bytes()
	}
}

func (m *ChangeMaplePointDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.sn = r.ReadInt64()
		m.count = r.ReadInt32()
	}
}
