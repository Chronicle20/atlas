package handler

import (
	"atlas-channel/character"
	_map "atlas-channel/map"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterDamageHandle = "CharacterDamageHandle"

func CharacterDamageHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		di := model.NewDamageTakenInfo(s.CharacterId())
		di.Decode(l, t, readerOptions)(r)

		// TODO process mana reflection
		// TODO process achilles
		// TODO process combo barrier
		// TODO process Body Pressure
		// TODO process PowerGuard
		// TODO process Paladin Divine Shield
		// TODO process Aran High Defense
		// TODO process MagicGuard
		// TODO process MesoGuard
		// TODO decrease battleship hp

		c, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
		if err != nil {
			return
		}

		err = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Map(), s.CharacterId(), session.Announce(l)(ctx)(wp)(writer.CharacterDamage)(writer.CharacterDamageBody(l)(ctx)(c, *di)))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce character [%d] has been damaged to foreign characters in map [%d].", s.CharacterId(), s.MapId())
		}

		_ = character.NewProcessor(l, ctx).ChangeHP(s.Map(), s.CharacterId(), -int16(di.Damage()))
	}
}
