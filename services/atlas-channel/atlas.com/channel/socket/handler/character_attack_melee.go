package handler

import (
	"atlas-channel/monster"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterMeleeAttackHandle = "CharacterMeleeAttackHandle"

func CharacterMeleeAttackHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		at := model2.NewAttackInfo(model2.AttackTypeMelee)
		at.Decode(l, t, readerOptions)(r)
		l.Debugf("Character [%d] is attempting a melee attack.", s.CharacterId())

		// TODO validate attack

		for _, di := range at.DamageInfo() {
			for _, d := range di.Damages() {
				err := monster.Damage(l)(ctx)(s.WorldId(), s.ChannelId(), di.MonsterId(), s.CharacterId(), d)
				if err != nil {
					l.WithError(err).Errorf("Unable to apply damage [%d] to monster [%d] from character [%d].", d, di.MonsterId(), s.CharacterId())
				}
			}
		}
	}
}
