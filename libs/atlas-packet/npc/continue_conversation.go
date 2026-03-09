package npc

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const NPCContinueConversationHandle = "NPCContinueConversationHandle"

type ContinueConversation struct {
	lastMessageType byte
	action          byte
}

func (m ContinueConversation) LastMessageType() byte {
	return m.lastMessageType
}

func (m ContinueConversation) Action() byte {
	return m.action
}

func (m ContinueConversation) Operation() string {
	return NPCContinueConversationHandle
}

func (m ContinueConversation) String() string {
	return fmt.Sprintf("lastMessageType [%d] action [%d]", m.lastMessageType, m.action)
}

func (m ContinueConversation) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.lastMessageType)
		w.WriteByte(m.action)
		return w.Bytes()
	}
}

func (m *ContinueConversation) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.lastMessageType = r.ReadByte()
		m.action = r.ReadByte()
	}
}
