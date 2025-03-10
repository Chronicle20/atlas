package handler

import (
	"atlas-channel/kafka/producer"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const PetMovementHandle = "PetMovementHandle"

func PetMovementHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		petId := r.ReadUint64()

		mp := model.Movement{}
		mp.Decode(l, t, readerOptions)(r)
		if len(mp.Elements) == 0 {
			return
		}

		l.Debugf("Pet [%d] moved for [%d].", petId, s.CharacterId())

		err := producer.ProviderImpl(l)(ctx)(pet.EnvCommandMovement)(pet.Move(petId, s.Map(), s.CharacterId(), mp))
		if err != nil {
			l.WithError(err).Errorf("Unable to distribute pet movement to other services.")
		}
	}
}
