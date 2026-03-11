package inventory

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ChangeEntry is a single entry in a ChangeBatch. Each entry type knows how
// to encode its body (mode + type + slot + data) and report its equipment
// state change indicator (addMov).
type ChangeEntry interface {
	EncodeEntry(logrus.FieldLogger, context.Context) func(options map[string]interface{}) []byte
	EntryAddMov() int8
}

// DecodeChangeEntry reads a single entry from the reader, dispatching by mode byte.
func DecodeChangeEntry(l logrus.FieldLogger, ctx context.Context, r *request.Reader, options map[string]interface{}) ChangeEntry {
	mode := ChangeMode(r.ReadByte())
	inventoryType := r.ReadByte()
	switch mode {
	case ChangeModeAdd:
		slot := r.ReadInt16()
		var a model.Asset
		a.Decode(l, ctx)(r, options)
		return AddEntry{inventoryType: inventoryType, slot: slot, asset: a}
	case ChangeModeQuantityUpdate:
		slot := r.ReadInt16()
		quantity := r.ReadUint16()
		return QuantityUpdateEntry{inventoryType: inventoryType, slot: slot, quantity: quantity}
	case ChangeModeMove:
		oldSlot := r.ReadInt16()
		newSlot := r.ReadInt16()
		return MoveEntry{inventoryType: inventoryType, oldSlot: oldSlot, newSlot: newSlot}
	case ChangeModeRemove:
		slot := r.ReadInt16()
		return RemoveEntry{inventoryType: inventoryType, slot: slot}
	default:
		return nil
	}
}

// AddEntry - mode(0) + inventoryType + slot + asset
type AddEntry struct {
	inventoryType byte
	slot          int16
	asset         model.Asset
}

func NewAddEntry(inventoryType byte, slot int16, asset model.Asset) AddEntry {
	return AddEntry{inventoryType: inventoryType, slot: slot, asset: asset}
}

func (m AddEntry) InventoryType() byte { return m.inventoryType }
func (m AddEntry) Slot() int16         { return m.slot }
func (m AddEntry) Asset() model.Asset  { return m.asset }
func (m AddEntry) EntryAddMov() int8   { return -1 }
func (m AddEntry) String() string      { return fmt.Sprintf("add entry slot [%d]", m.slot) }

func (m AddEntry) EncodeEntry(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(ChangeModeAdd))
		w.WriteByte(m.inventoryType)
		w.WriteInt16(m.slot)
		w.WriteByteArray(m.asset.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

// QuantityUpdateEntry - mode(1) + inventoryType + slot + quantity
type QuantityUpdateEntry struct {
	inventoryType byte
	slot          int16
	quantity      uint16
}

func NewQuantityUpdateEntry(inventoryType byte, slot int16, quantity uint16) QuantityUpdateEntry {
	return QuantityUpdateEntry{inventoryType: inventoryType, slot: slot, quantity: quantity}
}

func (m QuantityUpdateEntry) InventoryType() byte { return m.inventoryType }
func (m QuantityUpdateEntry) Slot() int16         { return m.slot }
func (m QuantityUpdateEntry) Quantity() uint16    { return m.quantity }
func (m QuantityUpdateEntry) EntryAddMov() int8   { return -1 }
func (m QuantityUpdateEntry) String() string {
	return fmt.Sprintf("quantity update entry slot [%d] quantity [%d]", m.slot, m.quantity)
}

func (m QuantityUpdateEntry) EncodeEntry(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(ChangeModeQuantityUpdate))
		w.WriteByte(m.inventoryType)
		w.WriteInt16(m.slot)
		w.WriteShort(m.quantity)
		return w.Bytes()
	}
}

// MoveEntry - mode(2) + inventoryType + oldSlot + newSlot
type MoveEntry struct {
	inventoryType byte
	oldSlot       int16
	newSlot       int16
}

func NewMoveEntry(inventoryType byte, oldSlot int16, newSlot int16) MoveEntry {
	return MoveEntry{inventoryType: inventoryType, oldSlot: oldSlot, newSlot: newSlot}
}

func (m MoveEntry) InventoryType() byte { return m.inventoryType }
func (m MoveEntry) OldSlot() int16      { return m.oldSlot }
func (m MoveEntry) NewSlot() int16      { return m.newSlot }
func (m MoveEntry) String() string {
	return fmt.Sprintf("move entry slot [%d] to [%d]", m.oldSlot, m.newSlot)
}

func (m MoveEntry) EntryAddMov() int8 {
	if m.inventoryType == 1 && m.newSlot < 0 {
		return 2
	} else if m.inventoryType == 1 && m.oldSlot < 0 {
		return 1
	}
	return -1
}

func (m MoveEntry) EncodeEntry(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(ChangeModeMove))
		w.WriteByte(m.inventoryType)
		w.WriteInt16(m.oldSlot)
		w.WriteInt16(m.newSlot)
		return w.Bytes()
	}
}

// RemoveEntry - mode(3) + inventoryType + slot
type RemoveEntry struct {
	inventoryType byte
	slot          int16
}

func NewRemoveEntry(inventoryType byte, slot int16) RemoveEntry {
	return RemoveEntry{inventoryType: inventoryType, slot: slot}
}

func (m RemoveEntry) InventoryType() byte { return m.inventoryType }
func (m RemoveEntry) Slot() int16         { return m.slot }
func (m RemoveEntry) String() string      { return fmt.Sprintf("remove entry slot [%d]", m.slot) }

func (m RemoveEntry) EntryAddMov() int8 {
	if m.inventoryType == 1 && m.slot < 0 {
		return 2
	}
	return -1
}

func (m RemoveEntry) EncodeEntry(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(ChangeModeRemove))
		w.WriteByte(m.inventoryType)
		w.WriteInt16(m.slot)
		return w.Bytes()
	}
}
