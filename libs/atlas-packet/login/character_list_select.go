package login

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterSelectedHandle = "CharacterSelectedHandle"

// CharacterSelected - CLogin::SendSelectCharPacket
type CharacterSelected struct {
	characterId uint32
	mac         string
	hwid        string
}

func (m CharacterSelected) CharacterId() uint32 {
	return m.characterId
}

func (m CharacterSelected) Mac() string {
	return m.mac
}

func (m CharacterSelected) Hwid() string {
	return m.hwid
}

func (m CharacterSelected) Operation() string {
	return CharacterSelectedHandle
}

func (m CharacterSelected) String() string {
	return fmt.Sprintf("characterId [%d], mac [%s], hwid [%s]", m.characterId, m.mac, m.hwid)
}

func (m CharacterSelected) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
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

func (m CharacterSelected) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		if t.Region() == "GMS" && t.MajorVersion() > 12 {
			m.mac = r.ReadAsciiString()
			m.hwid = r.ReadAsciiString()
		}
	}
}
