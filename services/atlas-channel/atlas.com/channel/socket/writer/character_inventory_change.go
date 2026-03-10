package writer

import (
	"atlas-channel/asset"
	model2 "atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	invpkt "github.com/Chronicle20/atlas-packet/inventory"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
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

func CharacterInventoryChangeBody(silent bool, writers ...InventoryChangeWriter) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			var entryBytes [][]byte
			addMov := int8(-1)
			for _, wf := range writers {
				w := response.NewWriter(l)
				tMov, _ := wf(w)
				entryBytes = append(entryBytes, w.Bytes())
				if tMov > -1 {
					addMov = tMov
				}
			}
			return invpkt.NewChangeBatch(silent, entryBytes, addMov).Encode(l, ctx)(options)
		}
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

func CharacterInventoryRefreshAsset(inventoryType inventory.Type, a asset.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			writers := make([]InventoryChangeWriter, 0)
			writers = append(writers, InventoryAddBodyWriter(inventoryType, a.Slot(), func(writer *response.Writer) error {
				return model2.NewAssetWriter(l, ctx, options, writer)(true)(a)
			}))
			return CharacterInventoryChangeBody(false, writers...)(l, ctx)(options)
		}
	}
}
