package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const AdminChatHandle = "AdminChat"

// AdminChat - CField::SendChatMsgSlash#AdminChat (opcode varies per version).
// Sent by the /-command parser for admin chat commands. Body (uniform across
// every send-site and version): byte type, byte flag, string message.
type AdminChat struct {
	chatType byte
	flag     byte
	message  string
}

func NewAdminChat(chatType byte, flag byte, message string) AdminChat {
	return AdminChat{chatType: chatType, flag: flag, message: message}
}

func (m AdminChat) ChatType() byte    { return m.chatType }
func (m AdminChat) Flag() byte        { return m.flag }
func (m AdminChat) Message() string   { return m.message }

func (m AdminChat) Operation() string {
	return AdminChatHandle
}

func (m AdminChat) String() string {
	return fmt.Sprintf("chatType [%d], flag [%d], message [%s]", m.chatType, m.flag, m.message)
}

func (m AdminChat) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.chatType)
		w.WriteByte(m.flag)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *AdminChat) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.chatType = r.ReadByte()
		m.flag = r.ReadByte()
		m.message = r.ReadAsciiString()
	}
}
