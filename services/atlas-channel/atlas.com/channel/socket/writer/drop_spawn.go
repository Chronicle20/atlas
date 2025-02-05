package writer

import (
	"atlas-channel/drop"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const DropSpawn = "DropSpawn"

func DropSpawnBody(l logrus.FieldLogger, t tenant.Model) func(d drop.Model, delay int16) BodyProducer {
	return func(d drop.Model, delay int16) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(d.Mod())
			w.WriteInt(d.Id())
			w.WriteBool(d.Meso() > 0)
			w.WriteInt(d.ItemId())
			w.WriteInt(d.Owner())
			w.WriteByte(d.Type())
			w.WriteInt16(d.X())
			w.WriteInt16(d.Y())
			w.WriteInt(d.DropperId())
			if d.Mod() != 2 {
				w.WriteInt16(d.DropperX())
				w.WriteInt16(d.DropperY())
				w.WriteInt16(delay)
			}
			if d.Meso() == 0 {
				w.WriteInt64(-1)
			}
			if d.CharacterDrop() {
				w.WriteBool(false)
			} else {
				w.WriteBool(true)
			}
			return w.Bytes()
		}
	}
}
