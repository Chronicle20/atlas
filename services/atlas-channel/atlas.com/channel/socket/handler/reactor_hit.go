package handler

import (
	"atlas-channel/reactor"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	reactor2 "github.com/Chronicle20/atlas/libs/atlas-packet/reactor/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func ReactorHitHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := reactor2.HitRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		bMoveAction := p.DwHitOption() & 1
		direction := (p.DwHitOption() >> 1) & 1
		stance := uint16(bMoveAction) | uint16(direction<<1)
		err := reactor.NewProcessor(l, ctx).Hit(s.Field(), p.Oid(), s.CharacterId(), stance, p.SkillId())
		if err != nil {
			l.WithError(err).Errorf("Unable to send hit command for reactor [%d].", p.Oid())
		}
	}
}
