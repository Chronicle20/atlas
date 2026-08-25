package handler

import (
	"atlas-channel/monster"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func MobBanishPlayerHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := serverbound.MobBanishPlayer{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		if err := monster.NewProcessor(l, ctx).Banish(s.Field(), s.CharacterId(), p.MobTemplateId()); err != nil {
			l.WithError(err).Warnf("Character [%d] not banished by template [%d].", s.CharacterId(), p.MobTemplateId())
		}
	}
}
