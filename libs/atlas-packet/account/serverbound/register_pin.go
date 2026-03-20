package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const RegisterPinHandle = "RegisterPinHandle"

// RegisterPin - CLogin::OnCheckPinCodeResult
type RegisterPin struct {
	pinInput bool
	pin      string
}

func (m RegisterPin) PinInput() bool {
	return m.pinInput
}

func (m RegisterPin) Pin() string {
	return m.pin
}

func (m RegisterPin) Operation() string {
	return RegisterPinHandle
}

func (m RegisterPin) String() string {
	return fmt.Sprintf("pinInput [%t], pin [%s]", m.pinInput, m.pin)
}

func (m RegisterPin) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.PinInput())
		if m.PinInput() {
			w.WriteAsciiString(m.Pin())
		}
		return w.Bytes()
	}
}

func (m *RegisterPin) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.pinInput = r.ReadBool()
		if m.pinInput {
			m.pin = r.ReadAsciiString()
		}
	}
}
