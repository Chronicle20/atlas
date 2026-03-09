package quest

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ActionComplete struct {
	npcId     uint32
	x         int16
	y         int16
	selection int32
	autoStart bool
}

func NewActionComplete(autoStart bool) *ActionComplete {
	return &ActionComplete{autoStart: autoStart}
}

func (m ActionComplete) NpcId() uint32    { return m.npcId }
func (m ActionComplete) X() int16         { return m.x }
func (m ActionComplete) Y() int16         { return m.y }
func (m ActionComplete) Selection() int32 { return m.selection }

func (m ActionComplete) Operation() string { return "ActionComplete" }

func (m ActionComplete) String() string {
	return fmt.Sprintf("npcId [%d] x [%d] y [%d] selection [%d] autoStart [%t]", m.npcId, m.x, m.y, m.selection, m.autoStart)
}

func (m ActionComplete) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.npcId)
		if m.autoStart {
			w.WriteInt16(m.x)
			w.WriteInt16(m.y)
		}
		w.WriteInt32(m.selection)
		return w.Bytes()
	}
}

func (m *ActionComplete) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.npcId = r.ReadUint32()
		if m.autoStart {
			m.x = r.ReadInt16()
			m.y = r.ReadInt16()
		} else {
			m.x = -1
			m.y = -1
		}
		m.selection = r.ReadInt32()
	}
}
