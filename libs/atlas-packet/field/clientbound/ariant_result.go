package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const AriantResultWriter = "AriantResult"

// packet-audit:fname CField::OnWarnMessage
type AriantResult struct {
	message string
}

func NewAriantResult(message string) AriantResult {
	return AriantResult{message: message}
}

func (m AriantResult) Message() string { return m.message }

func (m AriantResult) Operation() string { return AriantResultWriter }
func (m AriantResult) String() string {
	return fmt.Sprintf("message [%s]", m.message)
}

func (m AriantResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *AriantResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.message = r.ReadAsciiString()
	}
}
