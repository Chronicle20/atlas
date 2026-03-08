package login

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterViewAllSelectedHandle = "CharacterViewAllSelectedHandle"

// AllCharacterListSelect - CLogin::SendSelectCharPacketByVAC
type AllCharacterListSelect struct {
	characterId uint32
	worldId     world.Id
	mac         string
	hwid        string
}

func (m AllCharacterListSelect) CharacterId() uint32 {
	return m.characterId
}

func (m AllCharacterListSelect) WorldId() world.Id {
	return m.worldId
}

func (m AllCharacterListSelect) Mac() string {
	return m.mac
}

func (m AllCharacterListSelect) Hwid() string {
	return m.hwid
}

func (m AllCharacterListSelect) Operation() string {
	return CharacterViewAllSelectedHandle
}

func (m AllCharacterListSelect) String() string {
	return fmt.Sprintf("characterId [%d], worldId [%d], mac [%s], hwid [%s]", m.characterId, m.worldId, m.mac, m.hwid)
}

func (m AllCharacterListSelect) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.CharacterId())
		w.WriteInt(uint32(m.WorldId()))
		w.WriteAsciiString(m.Mac())
		w.WriteAsciiString(m.Hwid())
		return w.Bytes()
	}
}

func (m AllCharacterListSelect) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.worldId = world.Id(r.ReadByte())
		m.mac = r.ReadAsciiString()
		m.hwid = r.ReadAsciiString()
	}
}
