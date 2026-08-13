package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/chakra"
	skill2 "atlas-channel/character/skill"
	dataskill "atlas-channel/data/skill"
	"atlas-channel/effective_stats"
	_map "atlas-channel/map"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// CUserLocal::DoActiveSkill_Prepare (serverbound)
const CharacterSkillPrepareHandle = "CharacterSkillPrepareHandle"

// shouldBroadcastKeydown reports whether a keydown-skill prepare/cancel packet
// should be relayed to other sessions in the map.
//
// Conditions (D4):
//   - The character owns the skill (present in their skill book at level > 0).
//   - The skill resolves (via ctx's tenant version set) to a keydown-skill
//     Identity per IsKeyDownSkillIdentity. skillId is a raw wire id, and
//     keydown membership is version-scoped (task-187): wire 5101004 is the
//     keydown Brawler Corkscrew Blow at v0.62+ but the non-keydown SuperGM
//     Hide at v0.48.
func shouldBroadcastKeydown(ctx context.Context, skills []skill2.Model, skillId uint32) bool {
	t := tenant.MustFromContext(ctx)
	set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
	id, ok := set.Skill.Resolve(skill.Id(skillId))
	if !ok || !skill.IsKeyDownSkillIdentity(id) {
		return false
	}
	for _, sm := range skills {
		if sm.Id() == skill.Id(skillId) && sm.Level() > 0 {
			return true
		}
	}
	return false
}

// isChakraCast resolves a raw wire id to its version-blind Identity and
// reports whether it is Chief Bandit Chakra. Never compares the wire id
// directly (PRD FR-9.1).
func isChakraCast(ctx context.Context, skillId uint32) bool {
	t := tenant.MustFromContext(ctx)
	set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
	id, ok := set.Skill.Resolve(skill.Id(skillId))
	return ok && skill.IsIdentity(id, skill.ChiefBanditChakra)
}

// chakraPrepareDeps seams the three lookups the Chakra prepare gate needs so
// the gate itself is directly unit-testable without a live character,
// effective-stats or atlas-data service.
type chakraPrepareDeps struct {
	skillLevel     func() byte
	effectiveMaxHp func() uint32
	effectXY       func(level byte) (int16, int16, error)
	start          func(level byte, x int16, y int16)
}

// startChakraRecoveryWith is the activation gate (PRD FR-1). It runs at
// PREPARE time only; there is no post-gate HP re-check anywhere in the
// client (design §3.2) and PRD FR-1.3 forbids one server-side, so external
// healing that lifts the caster to >= 50% mid-window must not cancel the
// pending heal.
func startChakraRecoveryWith(l logrus.FieldLogger, hp uint16, baseMaxHp uint16, deps chakraPrepareDeps) {
	level := deps.skillLevel()
	if level == 0 {
		l.Debugf("Chakra prepare from a caster who does not own the skill; ignoring.")
		return
	}
	maxHp := chakra.EffectiveMaxHpOrBase(deps.effectiveMaxHp(), baseMaxHp)
	if !chakra.CanActivate(hp, maxHp) {
		l.Debugf("Chakra prepare rejected: hp [%d] is not below half of effective max hp [%d]. No window, no MP, no cooldown.", hp, maxHp)
		return
	}
	x, y, err := deps.effectXY(level)
	if err != nil {
		l.WithError(err).Warnf("Chakra prepare: unable to load the skill effect at level [%d]; no recovery window opened.", level)
		return
	}
	deps.start(level, x, y)
	l.Debugf("Chakra recovery window opened at level [%d] (x=[%d] damage-taken %%, y=[%d] recovery %%), hp [%d] of effective max [%d].", level, x, y, hp, maxHp)
}

// CharacterSkillPrepareHandleFunc handles the serverbound DoActiveSkill_Prepare
// packet. On validation pass it relays a foreign prepare packet to other map
// sessions. On miss it logs at debug level and returns (D3: no session destroy).
func CharacterSkillPrepareHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		info := packetmodel.NewSkillPrepareInfo()
		info.Decode(l, ctx)(r, readerOptions)

		cp := character.NewProcessor(l, ctx)
		c, err := cp.GetById(cp.SkillModelDecorator)(s.CharacterId())
		if err != nil {
			l.Debugf("Character [%d] skill prepare [%d]: character not found, skipping broadcast.", s.CharacterId(), info.SkillId())
			return
		}

		if isChakraCast(ctx, info.SkillId()) {
			t := tenant.MustFromContext(ctx)
			startChakraRecoveryWith(l, c.Hp(), c.MaxHp(), chakraPrepareDeps{
				skillLevel: func() byte { return skillLevelOf(c.Skills(), skill.Id(info.SkillId())) },
				effectiveMaxHp: func() uint32 {
					stats, sErr := effective_stats.NewProcessor(l, ctx).GetByCharacterId(s.Field().WorldId(), s.Field().ChannelId(), s.CharacterId())
					if sErr != nil {
						l.WithError(sErr).Warnf("Chakra prepare: effective stats unavailable for character [%d]; using base max hp.", s.CharacterId())
						return 0
					}
					return stats.MaxHp
				},
				effectXY: func(level byte) (int16, int16, error) {
					e, eErr := dataskill.NewProcessor(l, ctx).GetEffect(info.SkillId(), level)
					if eErr != nil {
						return 0, 0, eErr
					}
					return e.X(), e.Y(), nil
				},
				start: func(level byte, x int16, y int16) {
					chakra.GetRegistry().Start(t, s.CharacterId(), level, x, y, time.Now())
				},
			})
			// Chakra is NOT a keydown skill on any version (design §3.5:
			// is_keydown_skill excludes 4211001 on v83 and v95, reproducing
			// task-161). No foreign prepare broadcast.
			return
		}

		if !shouldBroadcastKeydown(ctx, c.Skills(), info.SkillId()) {
			l.Debugf("Character [%d] skill prepare [%d]: not a keydown skill or not owned, skipping broadcast.", s.CharacterId(), info.SkillId())
			return
		}

		_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), AnnounceForeignSkillPrepare(l)(ctx)(wp)(s.CharacterId(), *info))
	}
}
