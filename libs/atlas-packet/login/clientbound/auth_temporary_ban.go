package clientbound

import (
	"context"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const AuthTemporaryBanWriter = "AuthTemporaryBan"

type AuthTemporaryBan struct {
	bannedCode byte
	reason     byte
	until      uint64
}

func NewAuthTemporaryBan(bannedCode byte, reason byte, until time.Time) AuthTemporaryBan {
	return AuthTemporaryBan{bannedCode: bannedCode, reason: reason, until: uint64(msTime(until))}
}

func (m AuthTemporaryBan) BannedCode() byte { return m.bannedCode }
func (m AuthTemporaryBan) Reason() byte     { return m.reason }
func (m AuthTemporaryBan) Until() uint64    { return m.until }
func (m AuthTemporaryBan) Operation() string { return AuthTemporaryBanWriter }
func (m AuthTemporaryBan) String() string {
	return fmt.Sprintf("reason [%d]", m.reason)
}

func msTime(t time.Time) int64 {
	if t.IsZero() {
		return -1
	}
	return t.Unix()*int64(10000000) + int64(116444736000000000)
}

func (m AuthTemporaryBan) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.bannedCode)
		w.WriteByte(0)

		if t.Region() == "GMS" {
			w.WriteInt(0)
		}

		w.WriteByte(m.reason)
		w.WriteLong(m.until)
		return w.Bytes()
	}
}

func (m *AuthTemporaryBan) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.bannedCode = r.ReadByte()
		_ = r.ReadByte()

		if t.Region() == "GMS" {
			_ = r.ReadUint32()
		}

		m.reason = r.ReadByte()
		m.until = r.ReadUint64()
	}
}
