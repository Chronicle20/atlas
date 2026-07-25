package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// MountFoodHandle is the string handle the channel binds the tenant-configured
// taming-mob food opcode (0x4D, SendTamingMobFoodItemUseRequest) to.
const MountFoodHandle = "MountFoodHandle"

// Food is the serverbound taming-mob (mount) food packet body.
// v83 layout (IDA, context.md §2): ts(4), slot(2), itemId(4), all little-endian.
type Food struct {
	updateTime uint32
	slot       int16
	itemId     uint32
}

func (m Food) UpdateTime() uint32 {
	return m.updateTime
}

func (m Food) Slot() int16 {
	return m.slot
}

func (m Food) ItemId() uint32 {
	return m.itemId
}

func (m Food) Operation() string {
	return MountFoodHandle
}

func (m Food) String() string {
	return fmt.Sprintf("updateTime [%d] slot [%d] itemId [%d]", m.updateTime, m.slot, m.itemId)
}

func (m Food) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.slot)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *Food) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.slot = r.ReadInt16()
		m.itemId = r.ReadUint32()
	}
}
