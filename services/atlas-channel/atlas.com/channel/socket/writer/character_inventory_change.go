package writer

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
)

const CharacterInventoryChange = "CharacterInventoryChange"

type InventoryChangeMode byte

const (
	InventoryChangeModeAdd    = InventoryChangeMode(0)
	InventoryChangeModeUpdate = InventoryChangeMode(1)
	InventoryChangeModeMove   = InventoryChangeMode(2)
	InventoryChangeModeRemove = InventoryChangeMode(3)
)

func CharacterInventoryAddBody(inventoryType byte, slot int16, silent bool) func(itemWriter model.Operator[*response.Writer]) BodyProducer {
	return func(itemWriter model.Operator[*response.Writer]) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteBool(!silent)
			w.WriteByte(1) // size
			w.WriteByte(byte(InventoryChangeModeAdd))
			w.WriteByte(inventoryType)
			w.WriteInt16(slot)
			_ = itemWriter(w)
			return w.Bytes()
		}
	}
}

func CharacterInventoryUpdateBody(_ tenant.Model) func(inventoryType byte, slot int16, quantity uint32, silent bool) BodyProducer {
	return func(inventoryType byte, slot int16, quantity uint32, silent bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteBool(!silent)
			w.WriteByte(1) // size
			w.WriteByte(byte(InventoryChangeModeUpdate))
			w.WriteByte(inventoryType)
			w.WriteInt16(slot)
			w.WriteShort(uint16(quantity))
			return w.Bytes()
		}
	}
}

func CharacterInventoryMoveBody(_ tenant.Model) func(inventoryType byte, slot int16, oldSlot int16, silent bool) BodyProducer {
	return func(inventoryType byte, slot int16, oldSlot int16, silent bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteBool(!silent)
			w.WriteByte(1) // size
			w.WriteByte(byte(InventoryChangeModeMove))
			w.WriteByte(inventoryType)
			w.WriteInt16(oldSlot)
			w.WriteInt16(slot)
			addMovement := int8(-1)
			if slot < 0 {
				addMovement = 2
			} else if oldSlot < 0 {
				addMovement = 1
			}
			w.WriteInt8(addMovement)
			return w.Bytes()
		}
	}
}

func CharacterInventoryRemoveBody(_ tenant.Model) func(inventoryType byte, slot int16, silent bool) BodyProducer {
	return func(inventoryType byte, slot int16, silent bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteBool(!silent)
			w.WriteByte(1) // size
			w.WriteByte(byte(InventoryChangeModeRemove))
			w.WriteByte(inventoryType)
			w.WriteInt16(slot)
			w.WriteInt8(2)
			return w.Bytes()
		}
	}
}
