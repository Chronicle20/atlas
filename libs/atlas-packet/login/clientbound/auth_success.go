package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const AuthSuccessWriter = "AuthSuccess"

type AuthSuccess struct {
	accountId uint32
	name      string
	gender    byte
	usesPin   bool
	pic       string
}

func NewAuthSuccess(accountId uint32, name string, gender byte, usesPin bool, pic string) AuthSuccess {
	return AuthSuccess{accountId: accountId, name: name, gender: gender, usesPin: usesPin, pic: pic}
}

func (m AuthSuccess) AccountId() uint32 { return m.accountId }
func (m AuthSuccess) Name() string      { return m.name }
func (m AuthSuccess) Gender() byte      { return m.gender }
func (m AuthSuccess) UsesPin() bool     { return m.usesPin }
func (m AuthSuccess) Pic() string       { return m.pic }
func (m AuthSuccess) Operation() string { return AuthSuccessWriter }
func (m AuthSuccess) String() string {
	return fmt.Sprintf("accountId [%d], name [%s], gender [%d]", m.accountId, m.name, m.gender)
}

func (m AuthSuccess) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(0) // success
		w.WriteByte(0)

		if t.Region() == "GMS" {
			w.WriteInt(0)
		}

		w.WriteInt(m.accountId)
		w.WriteByte(m.gender)
		w.WriteBool(false) // GM
		w.WriteByte(0)     // admin byte

		if t.Region() == "GMS" {
			if t.MajorVersion() > 12 {
				w.WriteByte(0) // country code
			}
			w.WriteAsciiString(m.name)

			if t.MajorVersion() > 12 {
				w.WriteByte(0)  // quiet ban reason
				w.WriteByte(0)  // quiet ban
				w.WriteLong(0)  // quiet ban timestamp
				w.WriteLong(0)  // creation timestamp
				w.WriteInt(1)   // nNumOfCharacter
				w.WriteBool(!m.usesPin)
				var needsPic = byte(0)
				if m.pic != "" {
					needsPic = byte(1)
				}
				w.WriteByte(needsPic)
			} else {
				w.WriteLong(0)
				w.WriteLong(0)
				w.WriteLong(0)
			}

			if t.MajorVersion() >= 87 {
				w.WriteLong(0)
			}
		} else if t.Region() == "JMS" {
			w.WriteAsciiString(m.name)
			w.WriteAsciiString(m.name)
			w.WriteByte(0)
			w.WriteByte(0)
			w.WriteByte(0)
			w.WriteByte(0)
			w.WriteByte(0) // enables secure password
			w.WriteByte(0)
			w.WriteLong(0)
			w.WriteAsciiString(m.name)
		}
		return w.Bytes()
	}
}

func (m *AuthSuccess) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		_ = r.ReadByte() // success code
		_ = r.ReadByte()

		if t.Region() == "GMS" {
			_ = r.ReadUint32()
		}

		m.accountId = r.ReadUint32()
		m.gender = r.ReadByte()
		_ = r.ReadBool() // GM
		_ = r.ReadByte() // admin byte

		if t.Region() == "GMS" {
			if t.MajorVersion() > 12 {
				_ = r.ReadByte() // country code
			}
			m.name = r.ReadAsciiString()

			if t.MajorVersion() > 12 {
				_ = r.ReadByte()  // quiet ban reason
				_ = r.ReadByte()  // quiet ban
				_ = r.ReadUint64() // quiet ban timestamp
				_ = r.ReadUint64() // creation timestamp
				_ = r.ReadUint32() // nNumOfCharacter
				pinDisabled := r.ReadBool()
				m.usesPin = !pinDisabled
				needsPic := r.ReadByte()
				if needsPic == 1 {
					m.pic = "set"
				}
			} else {
				_ = r.ReadUint64()
				_ = r.ReadUint64()
				_ = r.ReadUint64()
			}

			if t.MajorVersion() >= 87 {
				_ = r.ReadUint64()
			}
		} else if t.Region() == "JMS" {
			m.name = r.ReadAsciiString()
			_ = r.ReadAsciiString() // name2
			_ = r.ReadByte()
			_ = r.ReadByte()
			_ = r.ReadByte()
			_ = r.ReadByte()
			_ = r.ReadByte() // secure password
			_ = r.ReadByte()
			_ = r.ReadUint64()
			_ = r.ReadAsciiString() // name3
		}
	}
}
