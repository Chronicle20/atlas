package handler

import (
	"atlas-channel/movement"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/pet/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func PetMovementHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := serverbound.MovementRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		if len(p.MovementData().Elements) == 0 {
			return
		}

		// p.PetId() is the client's pet serial (GW_ItemSlotBase::liCashItemSN),
		// not the Atlas pet id that atlas-pets keys on.
		pm, err := pet.NewProcessor(l, ctx).GetBySerialNumber(s.CharacterId(), p.PetId())
		if err != nil {
			l.WithError(err).Debugf("Unable to resolve pet [%d] for character [%d].", p.PetId(), s.CharacterId())
			return
		}

		_ = movement.NewProcessor(l, ctx, wp).ForPet(s.Field(), s.CharacterId(), pm.Id(), p.MovementData())
	}
}
