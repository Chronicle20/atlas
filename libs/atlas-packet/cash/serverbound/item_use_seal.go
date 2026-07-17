package serverbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseSeal is the type-26/64/65 sub-body of the cash ItemUse packet (Sealing Locks).
type ItemUseSeal struct {
	inventoryType   int32
	slot            int32
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseSeal(updateTimeFirst bool) *ItemUseSeal {
	return &ItemUseSeal{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseSeal) InventoryType() int32 { return m.inventoryType }
func (m ItemUseSeal) Slot() int32          { return m.slot }
func (m ItemUseSeal) UpdateTime() uint32   { return m.updateTime }
func (m ItemUseSeal) Operation() string    { return "ItemUseSeal" }

func (m ItemUseSeal) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
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

func (m *ItemUseSeal) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.inventoryType = r.ReadInt32()
		m.slot = r.ReadInt32()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
