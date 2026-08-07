package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const HorntailCaveWriter = "HorntailCave"

// packet-audit:fname CField::OnHontailTimer
type HorntailCave struct {
	state   byte
	seconds uint32
}

func NewHorntailCave(state byte, seconds uint32) HorntailCave {
	return HorntailCave{state: state, seconds: seconds}
}

func (m HorntailCave) State() byte     { return m.state }
func (m HorntailCave) Seconds() uint32 { return m.seconds }

func (m HorntailCave) Operation() string { return HorntailCaveWriter }
func (m HorntailCave) String() string {
	return fmt.Sprintf("state [%d] seconds [%d]", m.state, m.seconds)
}

func (m HorntailCave) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.state)
		w.WriteInt(m.seconds)
		return w.Bytes()
	}
}

func (m *HorntailCave) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.state = r.ReadByte()
		m.seconds = r.ReadUint32()
	}
}
