package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const LoginHandle = "LoginHandle"

// Request - CLogin::SendCheckPasswordPacket
type Request struct {
	name           string
	password       string
	hwid           []byte
	gameRoomClient uint32
	gameStartMode  byte
	unknown1       byte
	unknown2       byte
}

func (m Request) Name() string {
	return m.name
}

func (m Request) Password() string {
	return m.password
}

func (m Request) HWID() []byte {
	return m.hwid
}

func (m Request) GameRoomClient() uint32 {
	return m.gameRoomClient
}

func (m Request) GameStartMode() byte {
	return m.gameStartMode
}

func (m Request) Operation() string {
	return LoginHandle
}

func (m Request) String() string {
	return fmt.Sprintf("name [%s], password [%s], gameRoomClient [%d], gameStartMode [%d]", m.name, m.password, m.gameRoomClient, m.gameStartMode)
}

func (m Request) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		w.WriteAsciiString(m.password)
		w.WriteByteArray(m.hwid)
		w.WriteInt(m.gameRoomClient)
		w.WriteByte(m.gameStartMode)
		w.WriteByte(m.unknown1)
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			w.WriteByte(m.unknown2)
		}
		return w.Bytes()
	}
}

func (m *Request) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
		m.password = r.ReadAsciiString()
		m.hwid = r.ReadBytes(16)
		m.gameRoomClient = r.ReadUint32()
		m.gameStartMode = r.ReadByte()
		m.unknown1 = r.ReadByte()
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.unknown2 = r.ReadByte()
		}
	}
}
