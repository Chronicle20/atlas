package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PinOperationWriter = "PinOperation"

type PinOperation struct {
	mode byte
}

func NewPinOperation(mode byte) PinOperation {
	return PinOperation{mode: mode}
}

func (m PinOperation) Mode() byte        { return m.mode }
func (m PinOperation) Operation() string  { return PinOperationWriter }
func (m PinOperation) String() string     { return fmt.Sprintf("mode [%d]", m.mode) }

func (m PinOperation) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *PinOperation) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
