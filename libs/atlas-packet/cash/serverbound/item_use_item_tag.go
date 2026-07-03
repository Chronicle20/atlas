package serverbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseItemTag is the type-25 sub-body of the cash ItemUse packet (Item Tag 5060000).
type ItemUseItemTag struct {
	slot            int16
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseItemTag(updateTimeFirst bool) *ItemUseItemTag {
	return &ItemUseItemTag{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseItemTag) Slot() int16        { return m.slot }
func (m ItemUseItemTag) UpdateTime() uint32 { return m.updateTime }
func (m ItemUseItemTag) Operation() string  { return "ItemUseItemTag" }

func (m ItemUseItemTag) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.slot)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseItemTag) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.slot = r.ReadInt16()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
