package writer

import (
	"atlas-channel/asset"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
)

const CharacterInventoryChange = "CharacterInventoryChange"

type InventoryChangeMode byte

const (
	InventoryChangeModeAdd            = InventoryChangeMode(0)
	InventoryChangeModeQuantityUpdate = InventoryChangeMode(1)
	InventoryChangeModeMove           = InventoryChangeMode(2)
	InventoryChangeModeRemove         = InventoryChangeMode(3)
	InventoryChangeModeUnk            = InventoryChangeMode(4)
)

type InventoryChangeWriter func(w *response.Writer) (int8, error)

func CharacterInventoryChangeBody(silent bool, writers ...InventoryChangeWriter) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteBool(!silent)
		w.WriteByte(byte(len(writers)))
		addMov := int8(-1)
		for _, wf := range writers {
			tMov, _ := wf(w)
			if tMov > -1 {
				addMov = tMov
			}
		}
		if addMov > -1 {
			w.WriteInt8(addMov)
		}
		return w.Bytes()
	}
}

func InventoryAddBodyWriter(inventoryType inventory.Type, slot int16, itemWriter model.Operator[*response.Writer]) InventoryChangeWriter {
	return func(w *response.Writer) (int8, error) {
		w.WriteByte(byte(InventoryChangeModeAdd))
		w.WriteByte(byte(inventoryType))
		w.WriteInt16(slot)
		err := itemWriter(w)
		return -1, err
	}
}

func InventoryQuantityUpdateBodyWriter(inventoryType inventory.Type, slot int16, quantity uint32) InventoryChangeWriter {
	return func(w *response.Writer) (int8, error) {
		w.WriteByte(byte(InventoryChangeModeQuantityUpdate))
		w.WriteByte(byte(inventoryType))
		w.WriteInt16(slot)
		w.WriteShort(uint16(quantity))
		return -1, nil
	}
}

func InventoryMoveBodyWriter(inventoryType inventory.Type, slot int16, oldSlot int16) InventoryChangeWriter {
	return func(w *response.Writer) (int8, error) {
		w.WriteByte(byte(InventoryChangeModeMove))
		w.WriteByte(byte(inventoryType))
		w.WriteInt16(oldSlot)
		w.WriteInt16(slot)
		if inventoryType == inventory.TypeValueEquip && slot < 0 {
			return 2, nil
		} else if inventoryType == inventory.TypeValueEquip && oldSlot < 0 {
			return 1, nil
		}
		return -1, nil
	}
}

func InventoryRemoveBodyWriter(inventoryType inventory.Type, slot int16) InventoryChangeWriter {
	return func(w *response.Writer) (int8, error) {
		w.WriteByte(byte(InventoryChangeModeRemove))
		w.WriteByte(byte(inventoryType))
		w.WriteInt16(slot)
		if inventoryType == inventory.TypeValueEquip && slot < 0 {
			return 2, nil
		}
		return -1, nil
	}
}

func CharacterInventoryRefreshAsset(t tenant.Model) func(inventoryType inventory.Type, a asset.Model[any]) BodyProducer {
	return func(inventoryType inventory.Type, a asset.Model[any]) BodyProducer {
		pw := model.FlipOperator(WriteAssetInfo(t)(true))(a)
		return CharacterInventoryChangeBody(false, InventoryRemoveBodyWriter(inventoryType, a.Slot()), InventoryAddBodyWriter(inventoryType, a.Slot(), pw))
	}
}
