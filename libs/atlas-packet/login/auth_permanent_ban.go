package login

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const AuthPermanentBanWriter = "AuthPermanentBan"

type AuthPermanentBan struct {
	bannedCode byte
}

func NewAuthPermanentBan(bannedCode byte) AuthPermanentBan {
	return AuthPermanentBan{bannedCode: bannedCode}
}

func (m AuthPermanentBan) BannedCode() byte  { return m.bannedCode }
func (m AuthPermanentBan) Operation() string  { return AuthPermanentBanWriter }
func (m AuthPermanentBan) String() string     { return fmt.Sprintf("bannedCode [%d]", m.bannedCode) }

func (m AuthPermanentBan) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.bannedCode)
		w.WriteByte(0)

		if t.Region() == "GMS" {
			w.WriteInt(0)
		}

		w.WriteByte(0) // reason
		w.WriteLong(0) // timestamp
		return w.Bytes()
	}
}

func (m *AuthPermanentBan) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.bannedCode = r.ReadByte()
		_ = r.ReadByte()

		if t.Region() == "GMS" {
			_ = r.ReadUint32()
		}

		_ = r.ReadByte()   // reason
		_ = r.ReadUint64() // timestamp
	}
}
