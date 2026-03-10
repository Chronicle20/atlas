package writer

import (
	droppkt "github.com/Chronicle20/atlas-packet/drop"
	"github.com/Chronicle20/atlas-socket/packet"
)

type DropDestroyType byte

const (
	DropDestroy              = "DropDestroy"
	DropDestroyTypeExpire    = DropDestroyType(0)
	DropDestroyTypeNone      = DropDestroyType(1)
	DropDestroyTypePickUp    = DropDestroyType(2)
	DropDestroyTypeUnk1      = DropDestroyType(3)
	DropDestroyTypeExplode   = DropDestroyType(4)
	DropDestroyTypePetPickUp = DropDestroyType(5)
)

func DropDestroyBody(dropId uint32, destroyType DropDestroyType, characterId uint32, petSlot int8) packet.Encode {
	return droppkt.NewDropDestroy(dropId, droppkt.DropDestroyType(destroyType), characterId, petSlot).Encode
}
