package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const QuestActionHandle = "QuestActionHandle"

type Action struct {
	action  byte
	questId uint16
}

func (m Action) ActionType() byte {
	return m.action
}

func (m Action) QuestId() uint16 {
	return m.questId
}

func (m Action) Operation() string {
	return QuestActionHandle
}

func (m Action) String() string {
	return fmt.Sprintf("action [%d] questId [%d]", m.action, m.questId)
}

func (m Action) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.action)
		w.WriteShort(m.questId)
		return w.Bytes()
	}
}

func (m *Action) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.action = r.ReadByte()
		m.questId = r.ReadUint16()
	}
}
