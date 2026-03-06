package writer

import (
	"atlas-channel/reactor"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	ReactorSpawn = "ReactorSpawn"
)

func ReactorSpawnBody(m reactor.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(m.Id())
			w.WriteInt(m.Classification())
			w.WriteInt8(m.State())
			w.WriteInt16(m.X())
			w.WriteInt16(m.Y())
			w.WriteByte(m.Direction())
			w.WriteAsciiString(m.Name())
			return w.Bytes()
		}
	}
}
