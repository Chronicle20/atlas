package handler

import (
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	character2 "github.com/Chronicle20/atlas-packet/character"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterHealOverTimeHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := character2.HealOverTime{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		if p.HP() != 0 {
			_ = character.NewProcessor(l, ctx).ChangeHP(s.Field(), s.CharacterId(), p.HP())
		}
		if p.MP() != 0 {
			_ = character.NewProcessor(l, ctx).ChangeMP(s.Field(), s.CharacterId(), p.MP())
		}
	}
}
