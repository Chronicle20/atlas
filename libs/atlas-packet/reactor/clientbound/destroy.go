package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ReactorDestroyWriter = "ReactorDestroy"

type Destroy struct {
	id    uint32
	state int8
	x     int16
	y     int16
}

func NewReactorDestroy(id uint32, state int8, x int16, y int16) Destroy {
	return Destroy{id: id, state: state, x: x, y: y}
}

func (m Destroy) Id() uint32      { return m.id }
func (m Destroy) State() int8     { return m.state }
func (m Destroy) X() int16        { return m.x }
func (m Destroy) Y() int16        { return m.y }
func (m Destroy) Operation() string { return ReactorDestroyWriter }
func (m Destroy) String() string {
	return fmt.Sprintf("id [%d], state [%d]", m.id, m.state)
}

func (m Destroy) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.id)
		w.WriteInt8(m.state)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		return w.Bytes()
	}
}

func (m *Destroy) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.id = r.ReadUint32()
		m.state = r.ReadInt8()
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
	}
}
