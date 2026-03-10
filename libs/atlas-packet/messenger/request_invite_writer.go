package messenger

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type RequestInviteW struct {
	mode        byte
	fromName    string
	messengerId uint32
}

func NewMessengerRequestInvite(mode byte, fromName string, messengerId uint32) RequestInviteW {
	return RequestInviteW{mode: mode, fromName: fromName, messengerId: messengerId}
}

func (m RequestInviteW) Mode() byte           { return m.mode }
func (m RequestInviteW) FromName() string      { return m.fromName }
func (m RequestInviteW) MessengerId() uint32   { return m.messengerId }

func (m RequestInviteW) Operation() string { return MessengerOperationWriter }

func (m RequestInviteW) String() string {
	return fmt.Sprintf("messenger request invite from [%s] messengerId [%d]", m.fromName, m.messengerId)
}

func (m RequestInviteW) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.fromName)
		w.WriteByte(0)
		w.WriteInt(m.messengerId)
		w.WriteByte(0)
		return w.Bytes()
	}
}

func (m *RequestInviteW) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.fromName = r.ReadAsciiString()
		_ = r.ReadByte() // always zero
		m.messengerId = r.ReadUint32()
		_ = r.ReadByte() // always zero
	}
}
