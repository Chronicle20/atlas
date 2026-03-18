package serverbound

import (
	"context"
	"fmt"
	"strings"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterChatMultiHandle = "CharacterChatMultiHandle"

type Multi struct {
	chatType   byte
	recipients []uint32
	chatText   string
}

func (m Multi) ChatType() byte {
	return m.chatType
}

func (m Multi) Recipients() []uint32 {
	return m.recipients
}

func (m Multi) ChatText() string {
	return m.chatText
}

func (m Multi) Operation() string {
	return CharacterChatMultiHandle
}

func (m Multi) String() string {
	rs := make([]string, len(m.recipients))
	for i, r := range m.recipients {
		rs[i] = fmt.Sprintf("%d", r)
	}
	return fmt.Sprintf("chatType [%d] recipients [%s] chatText [%s]", m.chatType, strings.Join(rs, ","), m.chatText)
}

func (m Multi) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.chatType)
		w.WriteByte(byte(len(m.recipients)))
		for _, r := range m.recipients {
			w.WriteInt(r)
		}
		w.WriteAsciiString(m.chatText)
		return w.Bytes()
	}
}

func (m *Multi) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.chatType = r.ReadByte()
		recipientCount := r.ReadByte()
		m.recipients = make([]uint32, recipientCount)
		for i := 0; i < int(recipientCount); i++ {
			m.recipients[i] = r.ReadUint32()
		}
		m.chatText = r.ReadAsciiString()
	}
}
