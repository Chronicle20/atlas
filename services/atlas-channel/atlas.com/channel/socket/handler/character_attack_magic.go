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

const CharacterMagicAttackHandle = "CharacterMagicAttackHandle"

func CharacterMagicAttackHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		at := model2.NewAttackInfo(model2.AttackTypeMagic)
		at.Decode(l, t, readerOptions)(r)
		l.Debugf("Character [%d] is attempting a magic attack.", s.CharacterId())
	}
}
