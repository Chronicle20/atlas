package writer

import (
	"atlas-channel/pet"
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetMovement = "PetMovement"

func PetMovementBody(p pet.Model, movement model.Movement) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(p.OwnerId())
			w.WriteInt8(p.Slot())
			w.WriteByteArray(movement.Encode(l, ctx)(options))
			return w.Bytes()
		}
	}
}
