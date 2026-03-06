package writer

import (
	"github.com/Chronicle20/atlas-packet/model"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterMovement = "CharacterMovement"

func CharacterMovementBody(characterId uint32, movement model.Movement) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			w.WriteByteArray(movement.Encode(l, ctx)(options))
			return w.Bytes()
		}
	}
}
