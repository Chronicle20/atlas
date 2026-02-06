package writer

import (
	"atlas-channel/pet"
	"atlas-channel/socket/model"

	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const PetMovement = "PetMovement"

func PetMovementBody(l logrus.FieldLogger, t tenant.Model) func(p pet.Model, movement model.Movement) BodyProducer {
	return func(p pet.Model, movement model.Movement) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(p.OwnerId())
			w.WriteInt8(p.Slot())
			movement.Encode(l, t, options)(w)
			return w.Bytes()
		}
	}
}
