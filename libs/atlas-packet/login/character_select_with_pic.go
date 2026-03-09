package login

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterSelectedPicHandle = "CharacterSelectedPicHandle"

// CharacterSelectWithPic - CLogin::SendSelectCharPacket
type CharacterSelectWithPic struct {
	pic         string
	characterId uint32
	mac         string
	hwid        string
}

func (m CharacterSelectWithPic) Pic() string {
	return m.pic
}

func (m CharacterSelectWithPic) CharacterId() uint32 {
	return m.characterId
}

func (m CharacterSelectWithPic) Mac() string {
	return m.mac
}

func (m CharacterSelectWithPic) Hwid() string {
	return m.hwid
}

func (m CharacterSelectWithPic) Operation() string {
	return CharacterSelectedPicHandle
}

func (m CharacterSelectWithPic) String() string {
	return fmt.Sprintf("pic [%s], characterId [%d], mac [%s], hwid [%s]", m.pic, m.characterId, m.mac, m.hwid)
}

func (m CharacterSelectWithPic) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.Pic())
		w.WriteInt(m.CharacterId())
		if t.Region() == "GMS" {
			w.WriteAsciiString(m.Mac())
			w.WriteAsciiString(m.Hwid())
		}
		return w.Bytes()
	}
}

func (m *CharacterSelectWithPic) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.pic = r.ReadAsciiString()
		m.characterId = r.ReadUint32()

		if t.Region() == "GMS" {
			m.mac = r.ReadAsciiString()  // sMacAddressWithHDDSerial
			m.hwid = r.ReadAsciiString() // sMacAddressWithHDDSerial2
		}
	}
}
