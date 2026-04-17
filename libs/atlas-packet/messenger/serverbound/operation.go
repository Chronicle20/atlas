package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MessengerOperationHandle = "MessengerOperationHandle"

type Operation struct {
	mode byte
}

func (m Operation) Mode() byte {
	return m.mode
}

func (m Operation) Operation() string {
	return MessengerOperationHandle
}

func (m Operation) String() string {
	return fmt.Sprintf("mode [%d]", m.mode)
}

func (m Operation) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *Operation) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
