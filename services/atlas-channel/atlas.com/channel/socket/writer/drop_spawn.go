package writer

import (
	"atlas-channel/drop"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type DropEnterType byte

const (
	DropSpawn              = "DropSpawn"
	DropEnterTypeFresh     = DropEnterType(1)
	DropEnterTypeExisting  = DropEnterType(2)
	DropEnterTypeDisappear = DropEnterType(3)
)

func DropSpawnBody(d drop.Model, enterType DropEnterType, delay int16) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(byte(enterType))
			w.WriteInt(d.Id())
			if d.Meso() > 0 {
				w.WriteBool(true)
				w.WriteInt(d.Meso())
			} else {
				w.WriteBool(false)
				w.WriteInt(d.ItemId())
			}
			w.WriteInt(d.Owner())
			w.WriteByte(d.Type())
			w.WriteInt16(d.X())
			w.WriteInt16(d.Y())
			w.WriteInt(d.DropperId())
			if enterType != 2 {
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
