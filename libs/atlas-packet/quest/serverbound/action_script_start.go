package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ActionScriptStart struct {
	npcId uint32
	x     int16
	y     int16
}

func (m ActionScriptStart) NpcId() uint32 { return m.npcId }
func (m ActionScriptStart) X() int16      { return m.x }
func (m ActionScriptStart) Y() int16      { return m.y }

func (m ActionScriptStart) Operation() string { return "ActionScriptStart" }

func (m ActionScriptStart) String() string {
	return fmt.Sprintf("npcId [%d] x [%d] y [%d]", m.npcId, m.x, m.y)
}

func (m ActionScriptStart) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.npcId)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		return w.Bytes()
	}
}

func (m *ActionScriptStart) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.npcId = r.ReadUint32()
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
	}
}
