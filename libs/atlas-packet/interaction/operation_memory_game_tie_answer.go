package interaction

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationMemoryGameTieAnswer struct {
	response bool
}

func (m OperationMemoryGameTieAnswer) Response() bool { return m.response }

func (m OperationMemoryGameTieAnswer) Operation() string { return "OperationMemoryGameTieAnswer" }

func (m OperationMemoryGameTieAnswer) String() string {
	return fmt.Sprintf("response [%v]", m.response)
}

func (m OperationMemoryGameTieAnswer) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.response)
		return w.Bytes()
	}
}

func (m *OperationMemoryGameTieAnswer) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.response = r.ReadBool()
	}
}
