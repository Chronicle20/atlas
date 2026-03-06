package writer

import (
	"atlas-channel/kite"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SpawnKite = "SpawnKite"

func SpawnKiteBody(m kite.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(m.Id())
			w.WriteInt(m.TemplateId())
			w.WriteAsciiString(m.Message())
			w.WriteAsciiString(m.Name())
			w.WriteInt16(m.X())
			w.WriteInt16(m.Type())
			return w.Bytes()
		}
	}
}
