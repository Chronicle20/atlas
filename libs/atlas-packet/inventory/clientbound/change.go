package clientbound

import (
	"context"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const InventoryChangeWriter = "CharacterInventoryChange"

// QuantityUpdate - silent, single quantity update
type QuantityUpdate struct {
	silent        bool
	inventoryType byte
	slot          int16
	quantity      uint16
}

func NewQuantityUpdate(silent bool, inventoryType byte, slot int16, quantity uint16) QuantityUpdate {
	return QuantityUpdate{silent: silent, inventoryType: inventoryType, slot: slot, quantity: quantity}
}

func (m QuantityUpdate) Silent() bool        { return m.silent }
func (m QuantityUpdate) InventoryType() byte { return m.inventoryType }
func (m QuantityUpdate) Slot() int16         { return m.slot }
func (m QuantityUpdate) Quantity() uint16    { return m.quantity }
func (m QuantityUpdate) Operation() string   { return InventoryChangeWriter }
func (m QuantityUpdate) String() string {
	return fmt.Sprintf("quantity update slot [%d] quantity [%d]", m.slot, m.quantity)
}

func (m QuantityUpdate) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(!m.silent)
		w.WriteByte(1) // count
		w.WriteByte(byte(inventory.ChangeModeQuantityUpdate))
		w.WriteByte(m.inventoryType)
		w.WriteInt16(m.slot)
		w.WriteShort(m.quantity)
		return w.Bytes()
	}
}

func (m *QuantityUpdate) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.silent = !r.ReadBool()
		_ = r.ReadByte() // count
		_ = r.ReadByte() // mode
		m.inventoryType = r.ReadByte()
		m.slot = r.ReadInt16()
		m.quantity = r.ReadUint16()
	}
}

// ChangeMove - silent, single move operation
type ChangeMove struct {
	silent        bool
	inventoryType byte
	oldSlot       int16
	newSlot       int16
}

func NewChangeMove(silent bool, inventoryType byte, oldSlot int16, newSlot int16) ChangeMove {
	return ChangeMove{silent: silent, inventoryType: inventoryType, oldSlot: oldSlot, newSlot: newSlot}
}

func (m ChangeMove) Silent() bool        { return m.silent }
func (m ChangeMove) InventoryType() byte { return m.inventoryType }
func (m ChangeMove) OldSlot() int16      { return m.oldSlot }
func (m ChangeMove) NewSlot() int16      { return m.newSlot }
func (m ChangeMove) Operation() string   { return InventoryChangeWriter }
func (m ChangeMove) String() string {
	return fmt.Sprintf("move slot [%d] to [%d]", m.oldSlot, m.newSlot)
}

func (m ChangeMove) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(!m.silent)
		w.WriteByte(1) // count
		w.WriteByte(byte(inventory.ChangeModeMove))
		w.WriteByte(m.inventoryType)
		w.WriteInt16(m.oldSlot)
		w.WriteInt16(m.newSlot)
		addMov := int8(-1)
		if m.inventoryType == 1 && m.newSlot < 0 { // equip type
			addMov = 2
		} else if m.inventoryType == 1 && m.oldSlot < 0 {
			addMov = 1
		}
		if addMov > -1 {
			w.WriteInt8(addMov)
		}
		return w.Bytes()
	}
}

func (m *ChangeMove) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.silent = !r.ReadBool()
		_ = r.ReadByte() // count
		_ = r.ReadByte() // mode
		m.inventoryType = r.ReadByte()
		m.oldSlot = r.ReadInt16()
		m.newSlot = r.ReadInt16()
		if m.inventoryType == 1 && (m.newSlot < 0 || m.oldSlot < 0) {
			_ = r.ReadInt8() // addMov
		}
	}
}

// Remove - silent, single remove operation
type Remove struct {
	silent        bool
	inventoryType byte
	slot          int16
}

func NewInventoryRemove(silent bool, inventoryType byte, slot int16) Remove {
	return Remove{silent: silent, inventoryType: inventoryType, slot: slot}
}

func (m Remove) Silent() bool        { return m.silent }
func (m Remove) InventoryType() byte { return m.inventoryType }
func (m Remove) Slot() int16         { return m.slot }
func (m Remove) Operation() string   { return InventoryChangeWriter }
func (m Remove) String() string {
	return fmt.Sprintf("remove slot [%d]", m.slot)
}

func (m Remove) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(!m.silent)
		w.WriteByte(1) // count
		w.WriteByte(byte(inventory.ChangeModeRemove))
		w.WriteByte(m.inventoryType)
		w.WriteInt16(m.slot)
		if m.inventoryType == 1 && m.slot < 0 { // equip type
			w.WriteInt8(2) // addMov
		}
		return w.Bytes()
	}
}

func (m *Remove) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.silent = !r.ReadBool()
		_ = r.ReadByte() // count
		_ = r.ReadByte() // mode
		m.inventoryType = r.ReadByte()
		m.slot = r.ReadInt16()
		if m.inventoryType == 1 && m.slot < 0 {
			_ = r.ReadInt8() // addMov
		}
	}
}

// Add - silent, single add operation with an asset
type Add struct {
	silent        bool
	inventoryType byte
	slot          int16
	asset         model.Asset
}

func NewInventoryAdd(silent bool, inventoryType byte, slot int16, asset model.Asset) Add {
	return Add{silent: silent, inventoryType: inventoryType, slot: slot, asset: asset}
}

func (m Add) Silent() bool        { return m.silent }
func (m Add) InventoryType() byte { return m.inventoryType }
func (m Add) Slot() int16         { return m.slot }
func (m Add) Asset() model.Asset  { return m.asset }
func (m Add) Operation() string   { return InventoryChangeWriter }
func (m Add) String() string {
	return fmt.Sprintf("add slot [%d]", m.slot)
}

func (m Add) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(!m.silent)
		w.WriteByte(1) // count
		w.WriteByte(byte(inventory.ChangeModeAdd))
		w.WriteByte(m.inventoryType)
		w.WriteInt16(m.slot)
		w.WriteByteArray(m.asset.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *Add) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.silent = !r.ReadBool()
		_ = r.ReadByte() // count
		_ = r.ReadByte() // mode
		m.inventoryType = r.ReadByte()
		m.slot = r.ReadInt16()
		m.asset = model.NewAsset(true, 0, 0, time.Time{})
		m.asset.Decode(l, ctx)(r, options)
	}
}
