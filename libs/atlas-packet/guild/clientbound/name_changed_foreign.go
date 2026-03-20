package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const GuildNameChangedWriter = "GuildNameChanged"

type ForeignNameChanged struct {
	characterId uint32
	name        string
}

func NewForeignNameChanged(characterId uint32, name string) ForeignNameChanged {
	return ForeignNameChanged{characterId: characterId, name: name}
}

func (m ForeignNameChanged) Operation() string { return GuildNameChangedWriter }
func (m ForeignNameChanged) String() string {
	return fmt.Sprintf("characterId [%d], name [%s]", m.characterId, m.name)
}

func (m ForeignNameChanged) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *ForeignNameChanged) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.name = r.ReadAsciiString()
	}
}
