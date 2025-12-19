package handler

import (
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterTouchAttackHandle = "CharacterTouchAttackHandle"

func CharacterTouchAttackHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		at := model2.NewAttackInfo(model2.AttackTypeEnergy)
		at.Decode(l, t, readerOptions)(r)
		l.Debugf("Character [%d] is attempting a touch attack.", s.CharacterId())
		err := processAttack(l)(ctx)(wp)(*at)(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to completely process character [%d] touch attack.", s.CharacterId())
		}
	}
}
