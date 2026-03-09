package handler

import (
	"atlas-channel/monster"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	character2 "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

//CMob::Update

func MonsterDamageFriendlyHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := character2.MonsterDamageFriendly{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = monster.NewProcessor(l, ctx).DamageFriendly(s.Field(), p.AttackedId(), p.ObserverId(), p.AttackerId())
	}
}
