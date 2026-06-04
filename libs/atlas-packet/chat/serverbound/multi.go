package serverbound

import (
	"context"
	"fmt"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterChatMultiHandle = "CharacterMultiChatHandle"

type Multi struct {
	updateTime uint32
	chatType   byte
	recipients []uint32
	chatText   string
}

func (m Multi) UpdateTime() uint32 {
	return m.updateTime
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

func (m Multi) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	hasUpdateTime := t.Region() == "GMS" && t.MajorVersion() >= 95
	return func(options map[string]interface{}) []byte {
		if hasUpdateTime {
			w.WriteInt(m.updateTime)
		}
		w.WriteByte(m.chatType)
		w.WriteByte(byte(len(m.recipients)))
		for _, r := range m.recipients {
			w.WriteInt(r)
		}
		w.WriteAsciiString(m.chatText)
		return w.Bytes()
	}
}

func (m *Multi) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	hasUpdateTime := t.Region() == "GMS" && t.MajorVersion() >= 95
	return func(r *request.Reader, options map[string]interface{}) {
		if hasUpdateTime {
			m.updateTime = r.ReadUint32()
		}
		m.chatType = r.ReadByte()
		recipientCount := r.ReadByte()
		m.recipients = make([]uint32, recipientCount)
		for i := 0; i < int(recipientCount); i++ {
			m.recipients[i] = r.ReadUint32()
		}
		m.chatText = r.ReadAsciiString()
	}
}
