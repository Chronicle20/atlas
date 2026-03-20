package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ReactorHitWriter = "ReactorHit"

type Hit struct {
	id        uint32
	state     int8
	x         int16
	y         int16
	direction uint16
	unk1      byte
	unk2      byte
}

func NewReactorHit(id uint32, state int8, x int16, y int16, direction uint16) Hit {
	return Hit{id: id, state: state, x: x, y: y, direction: direction, unk1: 0, unk2: 5}
}

func (m Hit) Operation() string { return ReactorHitWriter }
func (m Hit) String() string {
	return fmt.Sprintf("id [%d], state [%d]", m.id, m.state)
}

func (m Hit) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.id)
		w.WriteInt8(m.state)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		w.WriteShort(m.direction)
		w.WriteByte(m.unk1)
		w.WriteByte(m.unk2)
		return w.Bytes()
	}
}

func (m *Hit) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.id = r.ReadUint32()
		m.state = r.ReadInt8()
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
		m.direction = r.ReadUint16()
		m.unk1 = r.ReadByte()
		m.unk2 = r.ReadByte()
	}
}
