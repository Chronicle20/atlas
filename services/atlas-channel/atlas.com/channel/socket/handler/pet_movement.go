package handler

import (
	"atlas-channel/movement"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const PetMovementHandle = "PetMovementHandle"

func PetMovementHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		petId := r.ReadUint64()

		mp := model.Movement{}
		mp.Decode(l, ctx)(r, readerOptions)
		if len(mp.Elements) == 0 {
			return
		}

		_ = movement.NewProcessor(l, ctx, wp).ForPet(s.Field(), s.CharacterId(), uint32(petId), mp)
	}
}
