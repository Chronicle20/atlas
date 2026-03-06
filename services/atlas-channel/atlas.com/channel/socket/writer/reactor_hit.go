package writer

import (
	"atlas-channel/reactor"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	ReactorHit = "ReactorHit"
)

func ReactorHitBody(m reactor.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
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
