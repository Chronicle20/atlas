package writer

import (
	"atlas-channel/data/npc"

	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SpawnNPCRequestController = "SpawnNPCRequestController"

func SpawnNPCRequestControllerBody(l logrus.FieldLogger) func(npc npc.Model, miniMap bool) BodyProducer {
	return func(npc npc.Model, miniMap bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(1)
			w.WriteInt(npc.Id())
			w.WriteInt(npc.Template())
			w.WriteInt16(npc.X())
			w.WriteInt16(npc.CY())
			if npc.F() == 1 {
				w.WriteByte(0)
			} else {
				w.WriteByte(1)
			}
			w.WriteShort(npc.Fh())
			w.WriteInt16(npc.RX0())
			w.WriteInt16(npc.RX1())
			w.WriteBool(miniMap)
			return w.Bytes()
		}
	}
}
