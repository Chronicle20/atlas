package writer

import (
	"atlas-channel/socket/model"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterMovement = "CharacterMovement"

func CharacterMovementBody(l logrus.FieldLogger, t tenant.Model) func(characterId uint32, movement model.Movement) BodyProducer {
	return func(characterId uint32, movement model.Movement) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			movement.Encode(l, t, options)(w)
			return w.Bytes()
		}
	}
}
