package handler

import (
	"atlas-channel/drop"
	"atlas-channel/party"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	drop2 "github.com/Chronicle20/atlas-packet/drop/serverbound"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func DropPickUpHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := drop2.PickUp{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		var partyId uint32
		pa, err := party.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
		if err == nil {
			partyId = pa.Id()
		}

		_ = drop.NewProcessor(l, ctx).RequestReservation(s.Field(), p.DropId(), s.CharacterId(), partyId, p.X(), p.Y(), -1)
	}
}
