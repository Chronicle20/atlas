package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type BBSDeleteThread struct {
	threadId uint32
}

func (m BBSDeleteThread) ThreadId() uint32 {
	return m.threadId
}

func (m BBSDeleteThread) Operation() string {
	return "BBSDeleteThread"
}

func (m BBSDeleteThread) String() string {
	return fmt.Sprintf("threadId [%d]", m.threadId)
}

func (m BBSDeleteThread) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.threadId)
		return w.Bytes()
	}
}

func (m *BBSDeleteThread) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.threadId = r.ReadUint32()
	}
}
