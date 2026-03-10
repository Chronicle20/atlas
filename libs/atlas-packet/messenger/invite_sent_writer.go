package messenger

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type InviteSentW struct {
	mode    byte
	message string
	success bool
}

func NewMessengerInviteSent(mode byte, message string, success bool) InviteSentW {
	return InviteSentW{mode: mode, message: message, success: success}
}

func (m InviteSentW) Mode() byte      { return m.mode }
func (m InviteSentW) Message() string  { return m.message }
func (m InviteSentW) Success() bool    { return m.success }

func (m InviteSentW) Operation() string { return MessengerOperationWriter }

func (m InviteSentW) String() string {
	return fmt.Sprintf("messenger invite sent message [%s] success [%t]", m.message, m.success)
}

func (m InviteSentW) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		w.WriteBool(m.success)
		return w.Bytes()
	}
}

func (m *InviteSentW) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
		m.success = r.ReadBool()
	}
}
