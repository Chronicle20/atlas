package writer

import (
	"atlas-channel/character/inventory/item"
	"atlas-channel/pet"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/response"
)

const CharacterInventoryChange = "CharacterInventoryChange"

type InventoryChangeMode byte

const (
	InventoryChangeModeAdd    = InventoryChangeMode(0)
	InventoryChangeModeUpdate = InventoryChangeMode(1)
	InventoryChangeModeMove   = InventoryChangeMode(2)
	InventoryChangeModeRemove = InventoryChangeMode(3)
)

type InventoryChangeWriter model.Operator[*response.Writer]

func CharacterInventoryChangeBody(silent bool, writers ...InventoryChangeWriter) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteBool(!silent)
		w.WriteByte(byte(len(writers)))
		for _, wf := range writers {
			_ = wf(w)
		}
		return w.Bytes()
	}
}

func InventoryAddBodyWriter(inventoryType inventory.Type, slot int16, itemWriter model.Operator[*response.Writer]) InventoryChangeWriter {
	return func(w *response.Writer) error {
		w.WriteByte(byte(InventoryChangeModeAdd))
		w.WriteByte(byte(inventoryType))
		w.WriteInt16(slot)
		return itemWriter(w)
	}
}

func InventoryUpdateBodyWriter(inventoryType inventory.Type, slot int16, quantity uint32) InventoryChangeWriter {
	return func(w *response.Writer) error {
		w.WriteByte(byte(InventoryChangeModeUpdate))
		w.WriteByte(byte(inventoryType))
		w.WriteInt16(slot)
		w.WriteShort(uint16(quantity))
		return nil
	}
}

func InventoryMoveBodyWriter(inventoryType inventory.Type, slot int16, oldSlot int16) InventoryChangeWriter {
	return func(w *response.Writer) error {
		w.WriteByte(byte(InventoryChangeModeMove))
		w.WriteByte(byte(inventoryType))
		w.WriteInt16(oldSlot)
		w.WriteInt16(slot)
		addMovement := int8(-1)
		if slot < 0 {
			addMovement = 2
		} else if oldSlot < 0 {
			addMovement = 1
		}
		w.WriteInt8(addMovement)
		return nil
	}
}

func InventoryRemoveBodyWriter(inventoryType inventory.Type, slot int16) InventoryChangeWriter {
	return func(w *response.Writer) error {
		w.WriteByte(byte(InventoryChangeModeRemove))
		w.WriteByte(byte(inventoryType))
		w.WriteInt16(slot)
		return nil
	}
}

func CharacterInventoryRefreshPet(p pet.Model, i item.Model) BodyProducer {
	pw := model.FlipOperator(WritePetCashItemInfo(true)(p))(i)
	writers := []InventoryChangeWriter{
		InventoryRemoveBodyWriter(inventory.TypeValueCash, i.Slot()),
		InventoryAddBodyWriter(inventory.TypeValueCash, i.Slot(), pw),
	}
	return CharacterInventoryChangeBody(true, writers...)
}
