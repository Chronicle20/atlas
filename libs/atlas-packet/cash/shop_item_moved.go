package cash

import (
	"context"
	"fmt"

	packetmodel "github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// CashItemMovedToInventory - mode, slot, asset
type CashItemMovedToInventory struct {
	mode  byte
	slot  uint16
	asset packetmodel.Asset
}

func NewCashItemMovedToInventory(mode byte, slot uint16, asset packetmodel.Asset) CashItemMovedToInventory {
	return CashItemMovedToInventory{mode: mode, slot: slot, asset: asset}
}

func (m CashItemMovedToInventory) Mode() byte              { return m.mode }
func (m CashItemMovedToInventory) Slot() uint16            { return m.slot }
func (m CashItemMovedToInventory) Asset() packetmodel.Asset { return m.asset }
func (m CashItemMovedToInventory) Operation() string        { return CashShopOperationWriter }

func (m CashItemMovedToInventory) String() string {
	return fmt.Sprintf("cash item moved to inventory slot [%d]", m.slot)
}

func (m CashItemMovedToInventory) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteShort(m.slot)
		w.WriteByteArray(m.asset.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *CashItemMovedToInventory) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.slot = r.ReadUint16()
		m.asset.Decode(l, ctx)(r, options)
	}
}

// CashItemMovedToCashInventory - mode, item
type CashItemMovedToCashInventory struct {
	mode byte
	item CashInventoryItem
}

func NewCashItemMovedToCashInventory(mode byte, item CashInventoryItem) CashItemMovedToCashInventory {
	return CashItemMovedToCashInventory{mode: mode, item: item}
}

func (m CashItemMovedToCashInventory) Mode() byte              { return m.mode }
func (m CashItemMovedToCashInventory) Item() CashInventoryItem { return m.item }
func (m CashItemMovedToCashInventory) Operation() string       { return CashShopOperationWriter }

func (m CashItemMovedToCashInventory) String() string {
	return fmt.Sprintf("cash item moved to cash inventory templateId [%d]", m.item.TemplateId)
}

func (m CashItemMovedToCashInventory) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByteArray(m.item.EncodeBytes(l))
		return w.Bytes()
	}
}

func (m *CashItemMovedToCashInventory) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.item = decodeCashInventoryItemSkipPadding(r)
	}
}
