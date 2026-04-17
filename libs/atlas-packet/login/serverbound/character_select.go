package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterSelectedHandle = "CharacterSelectedHandle"

// CharacterSelect - CLogin::SendSelectCharPacket
type CharacterSelect struct {
	characterId uint32
	mac         string
	hwid        string
}

func (m CharacterSelect) CharacterId() uint32 {
	return m.characterId
}

func (m CharacterSelect) Mac() string {
	return m.mac
}

func (m CharacterSelect) Hwid() string {
	return m.hwid
}

func (m CharacterSelect) Operation() string {
	return CharacterSelectedHandle
}

func (m CharacterSelect) String() string {
	return fmt.Sprintf("characterId [%d], mac [%s], hwid [%s]", m.characterId, m.mac, m.hwid)
}

func (m CharacterSelect) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.CharacterId())
		if t.Region() == "GMS" && t.MajorVersion() > 12 {
			w.WriteAsciiString(m.Mac())
			w.WriteAsciiString(m.Hwid())
		}
		return w.Bytes()
	}
}

func (m *CharacterSelect) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		if t.Region() == "GMS" && t.MajorVersion() > 12 {
			m.mac = r.ReadAsciiString()
			m.hwid = r.ReadAsciiString()
		}
	}
}
