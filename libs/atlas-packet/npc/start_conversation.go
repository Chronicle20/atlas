package npc

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const NPCStartConversationHandle = "NPCStartConversationHandle"

type StartConversation struct {
	oid uint32
	x   int16
	y   int16
}

func (m StartConversation) Oid() uint32 {
	return m.oid
}

func (m StartConversation) X() int16 {
	return m.x
}

func (m StartConversation) Y() int16 {
	return m.y
}

func (m StartConversation) Operation() string {
	return NPCStartConversationHandle
}

func (m StartConversation) String() string {
	return fmt.Sprintf("oid [%d] x [%d] y [%d]", m.oid, m.x, m.y)
}

func (m StartConversation) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.oid)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		return w.Bytes()
	}
}

func (m *StartConversation) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.oid = r.ReadUint32()
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
	}
}
