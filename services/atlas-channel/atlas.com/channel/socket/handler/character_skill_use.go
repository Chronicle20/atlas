package handler

import (
	"atlas-channel/character"
	skill2 "atlas-channel/character/skill"
	skill3 "atlas-channel/data/skill"
	_map "atlas-channel/map"
	"atlas-channel/session"
	"atlas-channel/skill/handler"
	"atlas-channel/socket/writer"
	summoncmd "atlas-channel/summon"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/summon"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
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

		se, err := skill3.NewProcessor(l, ctx).GetEffect(sui.SkillId(), sui.SkillLevel())
		if err != nil {
			err = enableActions(l)(ctx)(wp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", statpkt.StatChangedWriter, s.CharacterId())
			}
			return
		}

		l.Debugf("Character [%d] using skill [%d] at level [%d].", s.CharacterId(), sui.SkillId(), sui.SkillLevel())

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
			if sui.SkillId() == uint32(skill.DarkKnightBeholderId) {
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

func enableActions(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
		return func(wp writer.Producer) func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)
		}
	}
}
