package login

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterViewAllHandle = "CharacterViewAllHandle"

type AllCharacterListRequest struct {
	gameStartMode  byte
	nexonPassport  string
	machineId      []byte
	gameRoomClient uint32
	gameStartMode2 byte
}

func (m AllCharacterListRequest) GameStartMode() byte {
	return m.gameStartMode
}

func (m AllCharacterListRequest) NexonPassport() string {
	return m.nexonPassport
}

func (m AllCharacterListRequest) MachineId() []byte {
	return m.machineId
}

func (m AllCharacterListRequest) GameRoomClient() uint32 {
	return m.gameRoomClient
}

func (m AllCharacterListRequest) GameStartMode2() byte {
	return m.gameStartMode2
}

func (m AllCharacterListRequest) Operation() string {
	return CharacterViewAllHandle
}

func (m AllCharacterListRequest) String() string {
	return fmt.Sprintf("gameStartMode [%d], nexonPassport [%s], machineId [%s], gameRoomClient [%d], gameStartMode2 [%d]", m.gameStartMode, m.nexonPassport, m.machineId, m.gameRoomClient, m.gameStartMode2)
}

func (m AllCharacterListRequest) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		// TODO verify this conditional is actually necessary
		if t.Region() == "GMS" && t.MajorVersion() > 83 {
			w.WriteByte(m.GameStartMode())
			w.WriteAsciiString(m.NexonPassport())
			w.WriteByteArray(m.MachineId())
			w.WriteInt(m.GameRoomClient())
			w.WriteByte(m.GameStartMode2())
		}
		return w.Bytes()
	}
}

func (m AllCharacterListRequest) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.Region() == "GMS" && t.MajorVersion() > 83 {
			// TODO verify this conditional is actually necessary
			m.gameStartMode = r.ReadByte()
			m.nexonPassport = r.ReadAsciiString()
			m.machineId = r.ReadBytes(16)
			m.gameRoomClient = r.ReadUint32()
			m.gameStartMode2 = r.ReadByte()
		}
	}
}
