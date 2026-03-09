package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const DeleteCharacterHandle = "DeleteCharacterHandle"

// DeleteCharacter - CLogin::SendDeleteCharPacket
type DeleteCharacter struct {
	verifyPic   bool
	pic         string
	dob         uint32
	characterId uint32
}

func (m DeleteCharacter) VerifyPic() bool {
	return m.verifyPic
}

func (m DeleteCharacter) Pic() string {
	return m.pic
}

func (m DeleteCharacter) Dob() uint32 {
	return m.dob
}

func (m DeleteCharacter) CharacterId() uint32 {
	return m.characterId
}

func (m DeleteCharacter) Operation() string {
	return DeleteCharacterHandle
}

func (m DeleteCharacter) String() string {
	return fmt.Sprintf("verifyPic [%t], pic [%s], dob [%d], characterId [%d]", m.verifyPic, m.pic, m.dob, m.characterId)
}

func (m DeleteCharacter) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if t.Region() == "GMS" && t.MajorVersion() > 82 {
			w.WriteAsciiString(m.Pic())
		} else if t.Region() == "GMS" {
			w.WriteInt(m.Dob())
		}
		w.WriteInt(m.CharacterId())
		return w.Bytes()
	}
}

func (m *DeleteCharacter) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.Region() == "GMS" && t.MajorVersion() > 82 {
			m.verifyPic = true
			m.pic = r.ReadAsciiString()
		} else if t.Region() == "GMS" {
			m.dob = r.ReadUint32()
		}
		m.characterId = r.ReadUint32()
	}
}
