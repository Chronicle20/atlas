package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SnowballStateWriter = "SnowballState"

// packet-audit:fname CField_SnowBall::OnSnowBallState
type SnowballState struct {
	state     byte
	leftSnow  uint32
	rightSnow uint32
	snowmanHp uint16
	position  byte
	x0        uint16
	x1        uint16
	x2        uint16
}

func NewSnowballState(state byte, leftSnow uint32, rightSnow uint32, snowmanHp uint16, position byte, x0 uint16, x1 uint16, x2 uint16) SnowballState {
	return SnowballState{state: state, leftSnow: leftSnow, rightSnow: rightSnow, snowmanHp: snowmanHp, position: position, x0: x0, x1: x1, x2: x2}
}

func (m SnowballState) State() byte       { return m.state }
func (m SnowballState) LeftSnow() uint32  { return m.leftSnow }
func (m SnowballState) RightSnow() uint32 { return m.rightSnow }
func (m SnowballState) SnowmanHp() uint16 { return m.snowmanHp }
func (m SnowballState) Position() byte    { return m.position }
func (m SnowballState) X0() uint16        { return m.x0 }
func (m SnowballState) X1() uint16        { return m.x1 }
func (m SnowballState) X2() uint16        { return m.x2 }

func (m SnowballState) Operation() string { return SnowballStateWriter }
func (m SnowballState) String() string {
	return fmt.Sprintf("state [%d] leftSnow [%d] rightSnow [%d] snowmanHp [%d] position [%d] x0 [%d] x1 [%d] x2 [%d]", m.state, m.leftSnow, m.rightSnow, m.snowmanHp, m.position, m.x0, m.x1, m.x2)
}

func (m SnowballState) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.state)
		w.WriteInt(m.leftSnow)
		w.WriteInt(m.rightSnow)
		w.WriteShort(m.snowmanHp)
		w.WriteByte(m.position)
		w.WriteShort(m.x0)
		w.WriteShort(m.x1)
		w.WriteShort(m.x2)
		return w.Bytes()
	}
}

func (m *SnowballState) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.state = r.ReadByte()
		m.leftSnow = r.ReadUint32()
		m.rightSnow = r.ReadUint32()
		m.snowmanHp = r.ReadUint16()
		m.position = r.ReadByte()
		m.x0 = r.ReadUint16()
		m.x1 = r.ReadUint16()
		m.x2 = r.ReadUint16()
	}
}
