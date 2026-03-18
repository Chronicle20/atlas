package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type BBSReplyThread struct {
	threadId uint32
	message  string
}

func (m BBSReplyThread) ThreadId() uint32 {
	return m.threadId
}

func (m BBSReplyThread) Message() string {
	return m.message
}

func (m BBSReplyThread) Operation() string {
	return "BBSReplyThread"
}

func (m BBSReplyThread) String() string {
	return fmt.Sprintf("threadId [%d] message [%s]", m.threadId, m.message)
}

func (m BBSReplyThread) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.threadId)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *BBSReplyThread) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.threadId = r.ReadUint32()
		m.message = r.ReadAsciiString()
	}
}
