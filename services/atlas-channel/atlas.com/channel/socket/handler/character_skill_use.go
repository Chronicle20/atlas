package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	skill2 "atlas-channel/character/skill"
	skill3 "atlas-channel/data/skill"
	buff2 "atlas-channel/kafka/message/buff"
	_map "atlas-channel/map"
	"atlas-channel/session"
	"atlas-channel/skill/handler"
	"atlas-channel/socket/writer"
	summoncmd "atlas-channel/summon"
	"context"
	"time"

	"github.com/sirupsen/logrus"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/summon"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// CUserLocal::DoActiveSkill_TownPortal
// CUserLocal::DoActiveSkill_StatChangeAdmin
// CUserLocal::DoActiveSkill_Heal
// CUserLocal::DoActiveSkill_Summon
// CUserLocal::TryDoingMonsterMagnet
// CUserLocal::DoActiveSkill_SmokeShell
// CUserLocal::DoActiveSkill_RecoveryAura
// CUserLocal::DoActiveSkill_Flying
// CUserLocal::DoActiveSkill_DamageMeter
// CUserLocal::SendSkillUseRequest
// sub_A3ED44
// CGrenade::SendTimeBombInfo

const CharacterUseSkillHandle = "CharacterUseSkillHandle"

func CharacterUseSkillHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		sui := &packetmodel.SkillUsageInfo{}
		sui.Decode(l, ctx)(r, readerOptions)

		cp := character.NewProcessor(l, ctx)
		c, err := cp.GetById(cp.SkillModelDecorator)(s.CharacterId())
		if err != nil {
			err = enableActions(l)(ctx)(wp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", statpkt.StatChangedWriter, s.CharacterId())
			}
			return
		}
		if c.Hp() == 0 {
			l.Warnf("Character [%d] attempting to use skill when dead.", s.CharacterId())
			err = enableActions(l)(ctx)(wp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", statpkt.StatChangedWriter, s.CharacterId())
			}
			return
		}

		var sm skill2.Model
		for _, rs := range c.Skills() {
			if rs.Id() == skill.Id(sui.SkillId()) {
				sm = rs
			}
		}
		if sm.Id() == 0 || sm.Level() == 0 || sm.Level() != sui.SkillLevel() {
			l.Debugf("Character [%d] attempting to use skill [%d] at level [%d], but they do not have it.", s.CharacterId(), sui.SkillId(), sui.SkillLevel())
			_ = session.NewProcessor(l, ctx).Destroy(s)
			return
		}

		// Battleship post-break cooldown is enforced server-side: the client
		// greys the icon, but a packet-editing client must not remount a
		// broken ship (FR-2.4). Zero extra round-trips — CooldownExpiresAt is
		// decorated onto the already-loaded skill model.
		if battleshipCastBlocked(sui.SkillId(), sm.CooldownExpiresAt(), time.Now()) {
			l.Debugf("Character [%d] attempting to cast battleship while on post-break cooldown (expires [%s]).", s.CharacterId(), sm.CooldownExpiresAt())
			err = enableActions(l)(ctx)(wp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", statpkt.StatChangedWriter, s.CharacterId())
			}
			return
		}

		se, err := skill3.NewProcessor(l, ctx).GetEffect(sui.SkillId(), sui.SkillLevel())
		if err != nil {
			err = enableActions(l)(ctx)(wp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", statpkt.StatChangedWriter, s.CharacterId())
			}
			return
		}

		l.Debugf("Character [%d] using skill [%d] at level [%d].", s.CharacterId(), sui.SkillId(), sui.SkillLevel())

		// Resolved once and reused below (task-187): HeroEnrage and
		// DarkKnightBeholder are routed through the resolved Identity rather
		// than a raw wire compare, for defensive consistency with the guard
		// that bans raw wire compares outside the resolver.
		t := tenant.MustFromContext(ctx)
		set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
		castId, castIdOk := set.Skill.Resolve(skill.Id(sui.SkillId()))

		// Enrage (Hero) requires and consumes the caster's combo orbs. Gate the
		// cast here — before the buff applies — on the caster being at their orb
		// cap ("max combo orbs"), reading the live count from atlas-buffs. A cast
		// below the cap is rejected (no buff applied); an eligible cast has its
		// orbs consumed after the buff applies below. Fails OPEN on a buff-read
		// error so a transient atlas-buffs hiccup never blocks a legitimate cast.
		consumeEnrageOrbs := false
		var enrageComboSource int32
		if castIdOk && skill.IsIdentity(castId, skill.HeroEnrage) {
			line, hasCombo := comboSkillIds(c.Skills())
			if !hasCombo {
				l.Debugf("Character [%d] cast Enrage without a Combo Attack skill; rejecting.", s.CharacterId())
				if aerr := enableActions(l)(ctx)(wp)(s); aerr != nil {
					l.WithError(aerr).Errorf("Unable to write [%s] for character [%d].", statpkt.StatChangedWriter, s.CharacterId())
				}
				return
			}
			buffs, berr := buff.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())
			if berr != nil {
				l.WithError(berr).Warnf("Enrage: unable to read combo orbs for character [%d]; skipping orb-cap gate.", s.CharacterId())
			} else if !comboAtOrbCap(line, buffs, skill3.NewProcessor(l, ctx).GetEffect) {
				l.Debugf("Character [%d] cast Enrage below max combo orbs; rejecting.", s.CharacterId())
				if aerr := enableActions(l)(ctx)(wp)(s); aerr != nil {
					l.WithError(aerr).Errorf("Unable to write [%s] for character [%d].", statpkt.StatChangedWriter, s.CharacterId())
				}
				return
			}
			consumeEnrageOrbs = true
			enrageComboSource = int32(line.comboId)
		}

		// Summon skills additionally request atlas-summons to create the
		// owner-bound summon. This runs alongside (not instead of) the normal
		// skill-effect application below so the buff/cooldown still apply.
		if summon.IsSummonSkill(sui.SkillId()) {
			// For a Beholder (1321007) the heal/buff snapshot is driven by the
			// caster's trained AURA_OF_THE_BEHOLDER (1320008) and
			// HEX_OF_THE_BEHOLDER (1320009) levels, read here from the caster's
			// skill book (c.Skills() — decorated above). Non-Beholder summons
			// send 0/0.
			var auraLevel, hexLevel byte
			if castIdOk && skill.IsIdentity(castId, skill.DarkKnightBeholder) {
				auraLevel = skillLevelOf(c.Skills(), skill.DarkKnightAuraOfTheBeholderId)
				hexLevel = skillLevelOf(c.Skills(), skill.DarkKnightHexOfTheBeholderId)
			}
			if serr := summoncmd.NewProcessor(l, ctx).Spawn(s.Field(), s.CharacterId(), sui.SkillId(), sui.SkillLevel(), c.X(), c.Y(), auraLevel, hexLevel); serr != nil {
				l.WithError(serr).Errorf("Unable to request summon spawn for character [%d] skill [%d].", s.CharacterId(), sui.SkillId())
			}
		}

		err = handler.UseSkill(l)(ctx)(wp, s.Field(), s.CharacterId(), *sui, se)
		if err != nil {
			l.WithError(err).Errorf("Character [%d] failed to use skill [%d].", s.CharacterId(), sui.SkillId())
			return
		}

		// Enrage passed its orb-cap gate above and its buff has now applied —
		// consume the combo orbs (reset the COMBO stat to 1) via the delta
		// command atlas-buffs owns.
		if consumeEnrageOrbs {
			if cerr := buff.NewProcessor(l, ctx).UpdateStatValue(s.Field(), s.CharacterId(), buff.StatValueUpdate{
				SourceId:  enrageComboSource,
				StatType:  string(charconst.TemporaryStatTypeCombo),
				Operation: buff2.StatOperationSet,
				Amount:    1,
			}); cerr != nil {
				l.WithError(cerr).Errorf("Enrage: failed to consume combo orbs for character [%d].", s.CharacterId())
			}
		}

		session.NewProcessor(l, ctx).IfPresentByCharacterId(s.Field().Channel())(s.CharacterId(), AnnounceSkillUse(l)(ctx)(wp)(sui.SkillId(), c.Level(), sui.SkillLevel()))

		_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), AnnounceForeignSkillUse(l)(ctx)(wp)(s.CharacterId(), sui.SkillId(), c.Level(), sui.SkillLevel()))

		err = enableActions(l)(ctx)(wp)(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to write [%s] for character [%d].", statpkt.StatChangedWriter, s.CharacterId())
		}
	}
}

// skillLevelOf returns the caster's trained level in the given skill from their
// decorated skill book, or 0 if they have not learned it. Used to resolve the
// Beholder's aura/hex levels (1320008/1320009) for the summon snapshot.
func skillLevelOf(skills []skill2.Model, id skill.Id) byte {
	for _, sm := range skills {
		if sm.Id() == id {
			return sm.Level()
		}
	}
	return 0
}

// battleshipCastBlocked reports whether a 5221006 cast must be rejected
// because the post-break cooldown is still running (FR-2.4). Scoped to
// battleship: a generic cast-time cooldown gate is out of scope here.
func battleshipCastBlocked(skillId uint32, cooldownExpiresAt time.Time, now time.Time) bool {
	return skill.Id(skillId) == skill.CorsairBattleshipId && now.Before(cooldownExpiresAt)
}
