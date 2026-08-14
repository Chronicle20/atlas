// Package recoveryaura implements the Evan Recovery Aura (22161003) cast: it
// places a server-side aura at the caster's feet that periodically restores
// MP to the caster's party members standing inside it.
package recoveryaura

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/party"
	"atlas-channel/skill/handler/mistcast"
	"atlas-channel/socket/writer"
	"context"

	mistmsg "atlas-channel/kafka/message/mist"
	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// Recovery Aura is USE_SKILL-delivered, so it registers on the use-skill
// registry: `GET /api/data/skills/22161003` serves damage 100, attackCount 1,
// mobCount 1 and prop 0 on every version that binds it (gms 84/87/92/95 and
// jms 185) -- i.e. every attack node is ABSENT. UseSkill charges its
// HPConsume (18->34), its equal MPConsume, and its 60s cooldown before the
// handler lookup, so this handler charges nothing itself.
func init() {
	channelhandler.Register(skill2.EvanStage8RecoveryAura, Apply)
}

var (
	loadCaster = mistcast.DefaultLoadCaster
	emitCreate = mistcast.DefaultEmitCreate
)

// loadPartyMemberIds returns the caster's online party member ids. Package
// var so tests can stub it.
//
// This is a CAST-TIME snapshot, carried on the CREATE command, because
// atlas-maps has no party client and giving it one would add a service edge
// for a rule nothing client-side evaluates. The cost is a staleness window
// bounded by the aura's fixed 30s lifetime: someone who joins the party
// mid-aura is not healed, someone who leaves still is.
//
// Smokescreen deliberately does NOT work this way -- the client independently
// evaluates live party membership for smoke at hit time
// (CAffectedAreaPool::IsSmokeAreaByPoint), so a server snapshot there would
// visibly disagree with the player's own screen.
var loadPartyMemberIds = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) []uint32 {
	p, err := party.NewProcessor(l, ctx).GetByMemberId(characterId)
	if err != nil {
		// No party is the common case for a soloing caster, not an error
		// worth an error-level line; the caster still gets their own aura.
		l.WithError(err).Debugf("Recovery Aura: no party for character [%d]; scoping the aura to the caster.", characterId)
		return nil
	}
	ids := make([]uint32, 0, len(p.Members()))
	for _, m := range p.Members() {
		if !m.Online() {
			continue
		}
		ids = append(ids, m.Id())
	}
	return ids
}

// withCaster guarantees the caster is in the snapshot, so a soloing Evan --
// or one whose party lookup failed -- still benefits from their own aura.
func withCaster(ids []uint32, characterId uint32) []uint32 {
	for _, id := range ids {
		if id == characterId {
			return ids
		}
	}
	return append(ids, characterId)
}

// Apply is the Recovery Aura handler installed in the per-skill use-skill
// registry.
//
// The per-tick magnitude is the WZ `x` node (38 at L1 rising to 80 at L15),
// restored as MP: the skill's served description names MP explicitly, and
// `hp`, `mp`, `hpR` and `mpR` are all 0 at every level on every version that
// binds it (task-218 design §1.3). `x` is treated as an absolute MP amount,
// consistent with every other `x` consumer in this service.
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer,
	f field.Model,
	characterId uint32,
	info packetmodel.SkillUsageInfo,
	e effect.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer,
		f field.Model,
		characterId uint32,
		info packetmodel.SkillUsageInfo,
		e effect.Model,
	) error {
		return func(
			_ writer.Producer,
			f field.Model,
			characterId uint32,
			info packetmodel.SkillUsageInfo,
			e effect.Model,
		) error {
			if e.X() <= 0 {
				l.Warnf("Recovery Aura: rejected cast by [%d] — no recovery magnitude (WZ x = %d).", characterId, e.X())
				return nil
			}
			return mistcast.Cast(l, ctx, f, characterId, skill2.Id(info.SkillId()), info.SkillLevel(), e,
				mistcast.Params{
					SkillName:      "Recovery Aura",
					TargetKind:     mistmsg.TargetKindCharacter,
					EffectKind:     mistmsg.EffectKindRecovery,
					TickMs:         mistcast.PlayerMistTickIntervalMs,
					RecoveryMp:     int32(e.X()),
					PartyMemberIds: withCaster(loadPartyMemberIds(l, ctx, characterId), characterId),
				},
				mistcast.Seams{LoadCaster: loadCaster, EmitCreate: emitCreate})
		}
	}
}
