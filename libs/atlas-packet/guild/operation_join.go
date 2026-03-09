package guild

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type Join struct {
	guildId     uint32
	characterId uint32
}

func (m Join) GuildId() uint32     { return m.guildId }
func (m Join) CharacterId() uint32 { return m.characterId }

func (m Join) Operation() string { return "Join" }

func (m Join) String() string {
	return fmt.Sprintf("guildId [%d] characterId [%d]", m.guildId, m.characterId)
}

func (m Join) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.guildId)
		w.WriteInt(m.characterId)
		return w.Bytes()
	}
}

func (m *Join) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.guildId = r.ReadUint32()
		m.characterId = r.ReadUint32()
	}
}
