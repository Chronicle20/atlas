package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
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
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(byte(destroyType))
			w.WriteInt(dropId)
			if destroyType >= 2 {
				w.WriteInt(characterId)
				if petSlot >= 0 {
					w.WriteByte(byte(petSlot))
				}
			}
			return w.Bytes()
		}
	}
}
