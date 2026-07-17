package serverbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseIncubator is the sub-body of the cash ItemUse packet for the Incubator.
// Kept as a distinct struct from ItemUseSeal: the two packets are distinct client
// ops that only coincidentally share a shape at v83; verification is per-op and
// may diverge at later versions.
type ItemUseIncubator struct {
	inventoryType   int32
	slot            int32
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseIncubator(updateTimeFirst bool) *ItemUseIncubator {
	return &ItemUseIncubator{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseIncubator) InventoryType() int32 { return m.inventoryType }
func (m ItemUseIncubator) Slot() int32          { return m.slot }
func (m ItemUseIncubator) UpdateTime() uint32   { return m.updateTime }
func (m ItemUseIncubator) Operation() string    { return "ItemUseIncubator" }

func (m ItemUseIncubator) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.inventoryType)
		w.WriteInt32(m.slot)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseIncubator) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.inventoryType = r.ReadInt32()
		m.slot = r.ReadInt32()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
