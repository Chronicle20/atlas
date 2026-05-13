package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
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

		// v95 client's OnCheckPasswordResult permanent-ban branch (resultCode == 27)
		// reads only the 3 leading fields and routes to a dialog; the trailing
		// reason+timestamp are wasted bytes on v95. Keep them for older versions.
		if !(t.Region() == "GMS" && t.MajorVersion() >= 95) {
			w.WriteByte(0) // reason
			w.WriteLong(0) // timestamp
		}
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

		if !(t.Region() == "GMS" && t.MajorVersion() >= 95) {
			_ = r.ReadByte()   // reason
			_ = r.ReadUint64() // timestamp
		}
	}
}
