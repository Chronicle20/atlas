package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterViewAllSelectedPicHandle = "CharacterViewAllSelectedPicHandle"

// AllCharacterListSelectWithPic - CLogin::SendSelectCharPacketByVAC
type AllCharacterListSelectWithPic struct {
	pic         string
	characterId uint32
	worldId     world.Id
	mac         string
	hwid        string
}

func (m AllCharacterListSelectWithPic) Pic() string {
	return m.pic
}

func (m AllCharacterListSelectWithPic) CharacterId() uint32 {
	return m.characterId
}

func (m AllCharacterListSelectWithPic) WorldId() world.Id {
	return m.worldId
}

func (m AllCharacterListSelectWithPic) Mac() string {
	return m.mac
}

func (m AllCharacterListSelectWithPic) Hwid() string {
	return m.hwid
}

func (m AllCharacterListSelectWithPic) Operation() string {
	return CharacterViewAllSelectedPicHandle
}

func (m AllCharacterListSelectWithPic) String() string {
	return fmt.Sprintf("pic [%s], characterId [%d], worldId [%d], mac [%s], hwid [%s]", m.pic, m.characterId, m.worldId, m.mac, m.hwid)
}

func (m AllCharacterListSelectWithPic) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.pic)
		w.WriteInt(m.characterId)
		w.WriteInt(uint32(m.worldId))
		w.WriteAsciiString(m.mac)
		w.WriteAsciiString(m.hwid)
		return w.Bytes()
	}
}

func (m *AllCharacterListSelectWithPic) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.pic = r.ReadAsciiString()
		m.characterId = r.ReadUint32()
		m.worldId = world.Id(r.ReadUint32())
		m.mac = r.ReadAsciiString()
		m.hwid = r.ReadAsciiString()
	}
}
