package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const AfterLoginHandle = "AfterLoginHandle"

// AfterLogin - CLogin::OnSetAccountResult - CLogin::OnCheckPinCodeResult - CLogin::OnCheckPasswordResult - CLogin::OnSelectWorldResult
type AfterLogin struct {
	pinMode byte
	opt2    byte // 0 in OnCheckPinCodeResult
	pin     string
}

func (m AfterLogin) PinMode() byte {
	return m.pinMode
}

func (m AfterLogin) Opt2() byte {
	return m.opt2
}

func (m AfterLogin) Pin() string {
	return m.pin
}

func (m AfterLogin) Operation() string {
	return AfterLoginHandle
}

func (m AfterLogin) String() string {
	return fmt.Sprintf("pinMode [%d], opt2 [%d], pin [%s]", m.pinMode, m.opt2, m.pin)
}

func (m AfterLogin) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.PinMode())
		if m.PinMode() > 0 {
			w.WriteByte(m.Opt2())
			w.WriteAsciiString(m.Pin())
		}
		return w.Bytes()
	}
}

func (m *AfterLogin) Decode(l logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.pinMode = r.ReadByte()
		if m.pinMode > 0 {
			m.opt2 = r.ReadByte()
			m.pin = r.ReadAsciiString()
		}
	}
}
