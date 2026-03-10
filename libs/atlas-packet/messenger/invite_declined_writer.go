package messenger

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type InviteDeclinedW struct {
	mode        byte
	message     string
	declineMode byte
}

func NewMessengerInviteDeclined(mode byte, message string, declineMode byte) InviteDeclinedW {
	return InviteDeclinedW{mode: mode, message: message, declineMode: declineMode}
}

func (m InviteDeclinedW) Mode() byte        { return m.mode }
func (m InviteDeclinedW) Message() string    { return m.message }
func (m InviteDeclinedW) DeclineMode() byte  { return m.declineMode }

func (m InviteDeclinedW) Operation() string { return MessengerOperationWriter }

func (m InviteDeclinedW) String() string {
	return fmt.Sprintf("messenger invite declined message [%s] declineMode [%d]", m.message, m.declineMode)
}

func (m InviteDeclinedW) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		w.WriteByte(m.declineMode)
		return w.Bytes()
	}
}

func (m *InviteDeclinedW) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
		m.declineMode = r.ReadByte()
	}
}
