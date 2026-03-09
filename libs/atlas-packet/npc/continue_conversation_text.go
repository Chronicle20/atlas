package npc

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type ContinueConversationText struct {
	text string
}

func (m ContinueConversationText) Text() string { return m.text }

func (m ContinueConversationText) Operation() string { return "ContinueConversationText" }

func (m ContinueConversationText) String() string {
	return fmt.Sprintf("text [%s]", m.text)
}

func (m ContinueConversationText) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.text)
		return w.Bytes()
	}
}

func (m *ContinueConversationText) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.text = r.ReadAsciiString()
	}
}
