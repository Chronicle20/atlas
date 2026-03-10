package login

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const LoginAuthWriter = "LoginAuth"

type LoginAuth struct {
	screen string
}

func NewLoginAuth(screen string) LoginAuth {
	return LoginAuth{screen: screen}
}

func (m LoginAuth) Screen() string      { return m.screen }
func (m LoginAuth) Operation() string   { return LoginAuthWriter }
func (m LoginAuth) String() string      { return fmt.Sprintf("screen [%s]", m.screen) }

func (m LoginAuth) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.screen)
		return w.Bytes()
	}
}

func (m *LoginAuth) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.screen = r.ReadAsciiString()
	}
}
