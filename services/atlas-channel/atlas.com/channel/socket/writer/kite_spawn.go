package writer

import (
	"atlas-channel/kite"

	fieldpkt "github.com/Chronicle20/atlas-packet/field"
	"github.com/Chronicle20/atlas-socket/packet"
)

const SpawnKite = "SpawnKite"

func SpawnKiteBody(m kite.Model) packet.Encode {
	return fieldpkt.NewKiteSpawn(m.Id(), m.TemplateId(), m.Message(), m.Name(), m.X(), m.Type()).Encode
}
