package handler

import (
	"atlas-channel/drop"
	"atlas-channel/party"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	pet2 "github.com/Chronicle20/atlas/libs/atlas-packet/pet/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func PetDropPickUpHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		pk := pet2.DropPickUp{}
		pk.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", pk.Operation(), pk.String())

		p, err := pet.NewProcessor(l, ctx).GetById(uint32(pk.PetId()))
		if err != nil {
			l.WithError(err).Errorf("Unable to find pet [%d]", pk.PetId())
		}

		var partyId uint32
		pp, perr := party.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
		if perr == nil {
			partyId = pp.Id()
		}

		_ = drop.NewProcessor(l, ctx).RequestReservation(s.Field(), pk.DropId(), s.CharacterId(), partyId, pk.X(), pk.Y(), p.Slot())
	}
}
