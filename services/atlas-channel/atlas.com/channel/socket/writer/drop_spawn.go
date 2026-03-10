package writer

import (
	"atlas-channel/drop"

	droppkt "github.com/Chronicle20/atlas-packet/drop"
	"github.com/Chronicle20/atlas-socket/packet"
)

type DropEnterType byte

const (
	DropSpawn              = "DropSpawn"
	DropEnterTypeFresh     = DropEnterType(1)
	DropEnterTypeExisting  = DropEnterType(2)
	DropEnterTypeDisappear = DropEnterType(3)
)

func DropSpawnBody(d drop.Model, enterType DropEnterType, delay int16) packet.Encode {
	return droppkt.NewDropSpawn(
		droppkt.DropEnterType(enterType), d.Id(), d.Meso(), d.ItemId(),
		d.Owner(), d.Type(), d.X(), d.Y(), d.DropperId(),
		d.DropperX(), d.DropperY(), delay, d.CharacterDrop(),
	).Encode
}
