package drop

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const DropPickUpHandle = "DropPickUpHandle"

// PickUp - CUser::SendDropPickUpRequest
type PickUp struct {
	fieldKey   byte
	updateTime uint32
	x          int16
	y          int16
	dropId     uint32
	crc        uint32
}

func (m PickUp) FieldKey() byte {
	return m.fieldKey
}

func (m PickUp) UpdateTime() uint32 {
	return m.updateTime
}

func (m PickUp) X() int16 {
	return m.x
}

func (m PickUp) Y() int16 {
	return m.y
}

func (m PickUp) DropId() uint32 {
	return m.dropId
}

func (m PickUp) CRC() uint32 {
	return m.crc
}

func (m PickUp) Operation() string {
	return DropPickUpHandle
}

func (m PickUp) String() string {
	return fmt.Sprintf("fieldKey [%d], updateTime [%d], x [%d], y [%d], dropId [%d], crc [%d]", m.fieldKey, m.updateTime, m.x, m.y, m.dropId, m.crc)
}

func (m PickUp) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.fieldKey)
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		w.WriteInt(m.dropId)
		w.WriteInt(m.crc)
		return w.Bytes()
	}
}

func (m *PickUp) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.fieldKey = r.ReadByte()
		m.updateTime = r.ReadUint32()
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
		m.dropId = r.ReadUint32()
		m.crc = r.ReadUint32()
	}
}
