package handler

import (
	"atlas-channel/battleship"
	"atlas-channel/character"
	_map "atlas-channel/map"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"math"

	"github.com/sirupsen/logrus"

	atlaspacket "github.com/Chronicle20/atlas/libs/atlas-packet"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

		c, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
		if err != nil {
			return
		}

		err = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), session.Announce(l)(ctx)(wp)(charpkt.CharacterDamageWriter)(charpkt.NewCharacterDamage(c.Id(), p.AttackIdx(), p.Damage(), p.MonsterTemplateId(), p.Left()).Encode))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce character [%d] has been damaged to foreign characters in map [%d].", s.CharacterId(), s.MapId())
		}

		// Battleship: damage taken while riding drains the ship's parallel
		// HP pool (FR-3.1); the character HP change below is unaffected. A
		// non-breaking drain reports remaining ship HP via the skill-cooldown
		// packet carrying the config-resolved gauge pseudo-skill id
		// (FR-3.4 / DOM-25). Break (dismount + cooldown) is handled inside
		// Drain; the resulting client packets flow through the existing buff
		// and skill consumers.
		res := battleship.NewProcessor(l, ctx).Drain(s.Field(), s.CharacterId(), p.Damage())
		if res.Status == battleship.DrainDrained {
			announceShipHpGauge(l, ctx, wp, s, res.RemainingHP)
		}

		_ = character.NewProcessor(l, ctx).ChangeHP(s.Field(), s.CharacterId(), -int16(p.Damage()))
	}
}

// announceShipHpGauge sends the client's ship HP gauge: the skill-cooldown
// packet with the config-resolved battleship gauge pseudo-skill id and the
// remaining ship HP as the cooldown value (verified client behavior on
// v83/v84/v87/v95/jms185 — design §1.1). On any resolve miss the packet is
// skipped entirely (fail-loud, never send a guessed wire value).
func announceShipHpGauge(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, remaining int32) {
	t := tenant.MustFromContext(ctx)
	opts, ok := writer.TenantWriterOptions(t.Id(), charpkt.CharacterSkillCooldownWriter)
	if !ok {
		l.Errorf("Writer options for [%s] missing; battleship HP gauge not sent.", charpkt.CharacterSkillCooldownWriter)
		return
	}
	gaugeId, ok := atlaspacket.ResolveValue(l, opts, "skills", "BATTLESHIP_HP_GAUGE")
	if !ok {
		return
	}
	if err := session.Announce(l)(ctx)(wp)(charpkt.CharacterSkillCooldownWriter)(charpkt.NewCharacterSkillCooldown(gaugeId, gaugeCooldownValue(remaining)).Encode)(s); err != nil {
		l.WithError(err).Errorf("Unable to announce battleship HP gauge to character [%d].", s.CharacterId())
	}
}

// gaugeCooldownValue clamps remaining ship HP into the packet's uint16
// field. Battleship is maxLevel 10 on every version (R-5), so the ceiling is
// the v87+ arm at SLV 10 / charLevel 200 = 29 000 — well inside uint16. The
// clamp is purely defensive.
func gaugeCooldownValue(remaining int32) uint16 {
	if remaining < 0 {
		return 0
	}
	if remaining > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(remaining)
}
