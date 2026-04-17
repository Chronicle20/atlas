package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// CashInventoryItem represents a single item in the cash inventory.
type CashInventoryItem struct {
	CashId      int64
	AccountId   uint32
	CharacterId uint32
	TemplateId  uint32
	CommodityId uint32
	Quantity    int16
	GiftFrom    string
	Expiration  int64
}

func (m CashInventoryItem) EncodeBytes(l logrus.FieldLogger) []byte {
	w := response.NewWriter(l)
	w.WriteInt64(m.CashId)
	w.WriteInt(m.AccountId)
	w.WriteInt(m.CharacterId)
	w.WriteInt(m.TemplateId)
	w.WriteInt(m.CommodityId)
	w.WriteInt16(m.Quantity)
	model.WritePaddedString(w, m.GiftFrom, 13)
	w.WriteInt64(m.Expiration)
	w.WriteInt(0)
	w.WriteInt(0)
	return w.Bytes()
}

func DecodeCashInventoryItem(r *request.Reader) CashInventoryItem {
	return CashInventoryItem{
		CashId:      r.ReadInt64(),
		AccountId:   r.ReadUint32(),
		CharacterId: r.ReadUint32(),
		TemplateId:  r.ReadUint32(),
		CommodityId: r.ReadUint32(),
		Quantity:    r.ReadInt16(),
		GiftFrom:    model.ReadPaddedString(r, 13),
		Expiration:  r.ReadInt64(),
	}
}

func decodeCashInventoryItemSkipPadding(r *request.Reader) CashInventoryItem {
	item := DecodeCashInventoryItem(r)
	_ = r.ReadUint32() // padding
	_ = r.ReadUint32() // padding
	return item
}

// CashShopInventory - mode, items, storageSlots, characterSlots
type CashShopInventory struct {
	mode           byte
	items          []CashInventoryItem
	storageSlots   uint16
	characterSlots int16
}

func NewCashShopInventory(mode byte, items []CashInventoryItem, storageSlots uint16, characterSlots int16) CashShopInventory {
	return CashShopInventory{mode: mode, items: items, storageSlots: storageSlots, characterSlots: characterSlots}
}

func (m CashShopInventory) Mode() byte                  { return m.mode }
func (m CashShopInventory) Items() []CashInventoryItem   { return m.items }
func (m CashShopInventory) StorageSlots() uint16         { return m.storageSlots }
func (m CashShopInventory) CharacterSlots() int16        { return m.characterSlots }
func (m CashShopInventory) Operation() string            { return CashShopOperationWriter }

func (m CashShopInventory) String() string {
	return fmt.Sprintf("cash shop inventory items [%d]", len(m.items))
}

func (m CashShopInventory) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteShort(uint16(len(m.items)))
		for _, item := range m.items {
			w.WriteByteArray(item.EncodeBytes(l))
		}
		w.WriteShort(m.storageSlots)
		w.WriteInt16(m.characterSlots)
		return w.Bytes()
	}
}

func (m *CashShopInventory) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := int(r.ReadUint16())
		m.items = make([]CashInventoryItem, count)
		for i := 0; i < count; i++ {
			m.items[i] = decodeCashInventoryItemSkipPadding(r)
		}
		m.storageSlots = r.ReadUint16()
		m.characterSlots = r.ReadInt16()
	}
}

// CashShopPurchaseSuccess - mode, item
type CashShopPurchaseSuccess struct {
	mode byte
	item CashInventoryItem
}

func NewCashShopPurchaseSuccess(mode byte, item CashInventoryItem) CashShopPurchaseSuccess {
	return CashShopPurchaseSuccess{mode: mode, item: item}
}

func (m CashShopPurchaseSuccess) Mode() byte              { return m.mode }
func (m CashShopPurchaseSuccess) Item() CashInventoryItem { return m.item }
func (m CashShopPurchaseSuccess) Operation() string       { return CashShopOperationWriter }

func (m CashShopPurchaseSuccess) String() string {
	return fmt.Sprintf("cash shop purchase success templateId [%d]", m.item.TemplateId)
}

func (m CashShopPurchaseSuccess) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByteArray(m.item.EncodeBytes(l))
		return w.Bytes()
	}
}

func (m *CashShopPurchaseSuccess) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.item = decodeCashInventoryItemSkipPadding(r)
	}
}

// CashShopGifts - mode, empty gift list (stub)
type CashShopGifts struct {
	mode byte
}

func NewCashShopGifts(mode byte) CashShopGifts {
	return CashShopGifts{mode: mode}
}

func (m CashShopGifts) Mode() byte       { return m.mode }
func (m CashShopGifts) Operation() string { return CashShopOperationWriter }
func (m CashShopGifts) String() string    { return "cash shop gifts" }

func (m CashShopGifts) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteShort(0)
		return w.Bytes()
	}
}

func (m *CashShopGifts) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		_ = r.ReadUint16() // gift count (always 0 in current impl)
	}
}
