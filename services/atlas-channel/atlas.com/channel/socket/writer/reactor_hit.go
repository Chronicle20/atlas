package writer

import (
	"atlas-channel/reactor"

	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const (
	ReactorHit = "ReactorHit"
)

func ReactorHitBody(l logrus.FieldLogger, t tenant.Model) func(m reactor.Model) BodyProducer {
	return func(m reactor.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(m.Id())
			w.WriteInt8(m.State())
			w.WriteInt16(m.X())
			w.WriteInt16(m.Y())
			w.WriteShort(uint16(m.Direction()))
			w.WriteByte(0)
			w.WriteByte(5)
			return w.Bytes()
		}
	}
}
