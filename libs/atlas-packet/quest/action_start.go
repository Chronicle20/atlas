package quest

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ActionStart struct {
	npcId     uint32
	x         int16
	y         int16
	autoStart bool
}

func NewActionStart(autoStart bool) *ActionStart {
	return &ActionStart{autoStart: autoStart}
}

func (m ActionStart) NpcId() uint32 { return m.npcId }
func (m ActionStart) X() int16      { return m.x }
func (m ActionStart) Y() int16      { return m.y }

func (m ActionStart) Operation() string { return "ActionStart" }

func (m ActionStart) String() string {
	return fmt.Sprintf("npcId [%d] x [%d] y [%d] autoStart [%t]", m.npcId, m.x, m.y, m.autoStart)
}

func (m ActionStart) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.npcId)
		if m.autoStart {
			w.WriteInt16(m.x)
			w.WriteInt16(m.y)
		}
		return w.Bytes()
	}
}

func (m *ActionStart) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.npcId = r.ReadUint32()
		if m.autoStart {
			m.x = r.ReadInt16()
			m.y = r.ReadInt16()
		} else {
			m.x = -1
			m.y = -1
		}
	}
}
