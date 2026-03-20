package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterViewAllSelectedPicRegisterHandle = "CharacterViewAllSelectedPicRegisterHandle"

// AllCharacterListSelectWithPicRegister - CLogin::SendSelectCharPacketByVAC
type AllCharacterListSelectWithPicRegister struct {
	opt         byte
	characterId uint32
	worldId     world.Id
	mac         string
	hwid        string
	pic         string
}

func (m AllCharacterListSelectWithPicRegister) Opt() byte {
	return m.opt
}

func (m AllCharacterListSelectWithPicRegister) CharacterId() uint32 {
	return m.characterId
}

func (m AllCharacterListSelectWithPicRegister) WorldId() world.Id {
	return m.worldId
}

func (m AllCharacterListSelectWithPicRegister) Mac() string {
	return m.mac
}

func (m AllCharacterListSelectWithPicRegister) Hwid() string {
	return m.hwid
}

func (m AllCharacterListSelectWithPicRegister) Pic() string {
	return m.pic
}

func (m AllCharacterListSelectWithPicRegister) Operation() string {
	return CharacterViewAllSelectedPicRegisterHandle
}

func (m AllCharacterListSelectWithPicRegister) String() string {
	return fmt.Sprintf("opt [%d], characterId [%d], worldId [%d], mac [%s], hwid [%s], pic [%s]", m.opt, m.characterId, m.worldId, m.mac, m.hwid, m.pic)
}

func (m AllCharacterListSelectWithPicRegister) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.opt)
		w.WriteInt(m.characterId)
		w.WriteInt(uint32(m.worldId))
		w.WriteAsciiString(m.mac)
		w.WriteAsciiString(m.hwid)
		w.WriteAsciiString(m.pic)
		return w.Bytes()
	}
}

func (m *AllCharacterListSelectWithPicRegister) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.opt = r.ReadByte()
		m.characterId = r.ReadUint32()
		m.worldId = world.Id(r.ReadUint32())
		m.mac = r.ReadAsciiString()
		m.hwid = r.ReadAsciiString()
		m.pic = r.ReadAsciiString()
	}
}
