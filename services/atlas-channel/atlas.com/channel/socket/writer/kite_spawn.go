package writer

import (
	"atlas-channel/kite"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SpawnKite = "SpawnKite"

func SpawnKiteBody(l logrus.FieldLogger) func(m kite.Model) BodyProducer {
	return func(m kite.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
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
