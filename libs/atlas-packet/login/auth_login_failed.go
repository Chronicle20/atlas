package login

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const AuthLoginFailedWriter = "AuthLoginFailed"

type AuthLoginFailed struct {
	reason byte
}

func NewAuthLoginFailed(reason byte) AuthLoginFailed {
	return AuthLoginFailed{reason: reason}
}

func (m AuthLoginFailed) Reason() byte     { return m.reason }
func (m AuthLoginFailed) Operation() string { return AuthLoginFailedWriter }
func (m AuthLoginFailed) String() string    { return fmt.Sprintf("reason [%d]", m.reason) }

func (m AuthLoginFailed) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.reason)
		w.WriteByte(0)

		if t.Region() == "GMS" {
			w.WriteInt(0)
		}
		return w.Bytes()
	}
}

func (m *AuthLoginFailed) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.reason = r.ReadByte()
		_ = r.ReadByte()

		if t.Region() == "GMS" {
			_ = r.ReadUint32()
		}
	}
}
