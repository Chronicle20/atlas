package reactor

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ReactorSpawnWriter = "ReactorSpawn"

type Spawn struct {
	id             uint32
	classification uint32
	state          int8
	x              int16
	y              int16
	direction      byte
	name           string
}

func NewReactorSpawn(id uint32, classification uint32, state int8, x int16, y int16, direction byte, name string) Spawn {
	return Spawn{id: id, classification: classification, state: state, x: x, y: y, direction: direction, name: name}
}

func (m Spawn) Id() uint32              { return m.id }
func (m Spawn) Classification() uint32  { return m.classification }
func (m Spawn) State() int8             { return m.state }
func (m Spawn) X() int16                { return m.x }
func (m Spawn) Y() int16                { return m.y }
func (m Spawn) Direction() byte         { return m.direction }
func (m Spawn) Name() string            { return m.name }
func (m Spawn) Operation() string       { return ReactorSpawnWriter }
func (m Spawn) String() string {
	return fmt.Sprintf("id [%d], classification [%d], state [%d]", m.id, m.classification, m.state)
}

func (m Spawn) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.id)
		w.WriteInt(m.classification)
		w.WriteInt8(m.state)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		w.WriteByte(m.direction)
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *Spawn) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.id = r.ReadUint32()
		m.classification = r.ReadUint32()
		m.state = r.ReadInt8()
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
		m.direction = r.ReadByte()
		m.name = r.ReadAsciiString()
	}
}
