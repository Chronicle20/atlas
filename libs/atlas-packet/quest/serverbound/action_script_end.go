package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ActionScriptEnd struct {
	npcId uint32
	x     int16
	y     int16
}

func (m ActionScriptEnd) NpcId() uint32 { return m.npcId }
func (m ActionScriptEnd) X() int16      { return m.x }
func (m ActionScriptEnd) Y() int16      { return m.y }

func (m ActionScriptEnd) Operation() string { return "ActionScriptEnd" }

func (m ActionScriptEnd) String() string {
	return fmt.Sprintf("npcId [%d] x [%d] y [%d]", m.npcId, m.x, m.y)
}

func (m ActionScriptEnd) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.npcId)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		return w.Bytes()
	}
}

func (m *ActionScriptEnd) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.npcId = r.ReadUint32()
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
	}
}
