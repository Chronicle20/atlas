package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type BBSDisplayThread struct {
	threadId uint32
}

func (m BBSDisplayThread) ThreadId() uint32 {
	return m.threadId
}

func (m BBSDisplayThread) Operation() string {
	return "BBSDisplayThread"
}

func (m BBSDisplayThread) String() string {
	return fmt.Sprintf("threadId [%d]", m.threadId)
}

func (m BBSDisplayThread) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.threadId)
		return w.Bytes()
	}
}

func (m *BBSDisplayThread) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.threadId = r.ReadUint32()
	}
}
