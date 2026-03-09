package guild

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type BBSDeleteReply struct {
	threadId uint32
	replyId  uint32
}

func (m BBSDeleteReply) ThreadId() uint32 {
	return m.threadId
}

func (m BBSDeleteReply) ReplyId() uint32 {
	return m.replyId
}

func (m BBSDeleteReply) Operation() string {
	return "BBSDeleteReply"
}

func (m BBSDeleteReply) String() string {
	return fmt.Sprintf("threadId [%d] replyId [%d]", m.threadId, m.replyId)
}

func (m BBSDeleteReply) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.threadId)
		w.WriteInt(m.replyId)
		return w.Bytes()
	}
}

func (m *BBSDeleteReply) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.threadId = r.ReadUint32()
		m.replyId = r.ReadUint32()
	}
}
