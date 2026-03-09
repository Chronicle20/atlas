package login

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const RegisterPicHandle = "RegisterPicHandle"

// CharacterSelectRegisterPic - CLogin::SendSelectCharPacket
type CharacterSelectRegisterPic struct {
	mode        byte
	characterId uint32
	mac         string
	hwid        string
	pic         string
}

func (m CharacterSelectRegisterPic) Mode() byte {
	return m.mode
}

func (m CharacterSelectRegisterPic) CharacterId() uint32 {
	return m.characterId
}

func (m CharacterSelectRegisterPic) Mac() string {
	return m.mac
}

func (m CharacterSelectRegisterPic) Hwid() string {
	return m.hwid
}

func (m CharacterSelectRegisterPic) Pic() string {
	return m.pic
}

func (m CharacterSelectRegisterPic) Operation() string {
	return RegisterPicHandle
}

func (m CharacterSelectRegisterPic) String() string {
	return fmt.Sprintf("mode [%d], characterId [%d], mac [%s], hwid [%s], pic [%s]", m.mode, m.characterId, m.mac, m.hwid, m.pic)
}

func (m CharacterSelectRegisterPic) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.Mode())
		w.WriteInt(m.CharacterId())
		if t.Region() == "GMS" {
			w.WriteAsciiString(m.Mac())
			w.WriteAsciiString(m.Hwid())
		}
		w.WriteAsciiString(m.Pic())
		return w.Bytes()
	}
}

func (m CharacterSelectRegisterPic) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.characterId = r.ReadUint32()
		if t.Region() == "GMS" {
			m.mac = r.ReadAsciiString()  // sMacAddressWithHDDSerial - not logged for security
			m.hwid = r.ReadAsciiString() // sMacAddressWithHDDSerial2 - not logged for security
		}
		m.pic = r.ReadAsciiString()
	}
}
