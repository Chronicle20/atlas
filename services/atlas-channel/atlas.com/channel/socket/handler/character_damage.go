package handler

import (
	"atlas-channel/character"
	_map "atlas-channel/map"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	packetmodel "github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
	charpkt "github.com/Chronicle20/atlas-packet/character/clientbound"
)

func CharacterDamageHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := packetmodel.NewDamageTakenInfo(s.CharacterId())
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

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

		err = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), session.Announce(l)(ctx)(wp)(charpkt.CharacterDamageWriter)(charpkt.NewCharacterDamage(c.Id(), p.AttackIdx(), p.Damage(), p.MonsterTemplateId(), p.Left()).Encode))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce character [%d] has been damaged to foreign characters in map [%d].", s.CharacterId(), s.MapId())
		}

		_ = character.NewProcessor(l, ctx).ChangeHP(s.Field(), s.CharacterId(), -int16(p.Damage()))
	}
}
