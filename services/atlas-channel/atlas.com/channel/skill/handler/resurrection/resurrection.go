package resurrection

import (
	"context"
	"math"

	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	channelmap "atlas-channel/map"
	"atlas-channel/portal"
	"atlas-channel/session"
	channelhandler "atlas-channel/skill/handler"
	socketHandler "atlas-channel/socket/handler"
	"atlas-channel/socket/writer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

func init() {
	channelhandler.Register(skill2.BishopResurrectionId, Apply)
	channelhandler.Register(skill2.GmResurrectionId, Apply)
	channelhandler.Register(skill2.SuperGmResurrectionId, Apply)
}

// loadCaster fetches the caster's position (range-check origin for selectByVariant) and level
// (needed for the effect broadcast) in a single call. Seam for tests.
var loadCaster = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (int16, int16, byte, error) {
	c, err := character.NewProcessor(l, ctx).GetById()(characterId)
	if err != nil {
		return 0, 0, 0, err
	}
	return c.X(), c.Y(), c.Level(), nil
}

// setHP sends an absolute SET_HP command; atlas-character clamps to effective
// MaxHP, so math.MaxUint16 yields a full-HP restore. Seam for tests.
var setHP = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, amount uint16) error {
	return character.NewProcessor(l, ctx).SetHP(f, characterId, amount)
}

// warpToPosition warps a character to (x,y) on the current map via the task-093
// chase-warp primitive (a WarpToPosition command to atlas-portals). Per the
// task-111 design, an in-map warp to a dead character's own death coordinates is
// expected to make the v83 client play its revive animation; this is the OQ-1
// live-verification gate, not a property verified in this service.
var warpToPosition = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, x, y int16) error {
	return portal.NewProcessor(l, ctx).WarpToPosition(f, characterId, f.MapId(), x, y)
}

// broadcastEffects fires the holy-light skill-use effect to the caster and the
// foreign skill-use effect to other players in the map. Seam for tests.
var broadcastEffects = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, casterId uint32, casterLevel byte, skillId uint32, skillLevel byte) {
	_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(f.Channel())(
		casterId,
		socketHandler.AnnounceSkillUse(l)(ctx)(wp)(skillId, casterLevel, skillLevel),
	)
	_ = channelmap.NewProcessor(l, ctx).ForOtherSessionsInMap(
		f, casterId,
		socketHandler.AnnounceForeignSkillUse(l)(ctx)(wp)(casterId, skillId, casterLevel, skillLevel),
	)
}

// Apply is the Resurrection handler installed in the per-skill registry for the
// Bishop and GM/SuperGM skill IDs. For each dead recipient it restores full HP
// then warps the recipient to its own death coordinates (expected, per
// design/OQ-1, to trigger the client revive). Per-recipient failures are logged
// and skipped; caster-load failure is a clean no-op. An empty recipient set
// still broadcasts the effect.
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer, f field.Model, characterId uint32,
	info packetmodel.SkillUsageInfo, e effect.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer, f field.Model, characterId uint32,
		info packetmodel.SkillUsageInfo, e effect.Model,
	) error {
		return func(
			wp writer.Producer, f field.Model, characterId uint32,
			info packetmodel.SkillUsageInfo, e effect.Model,
		) error {
			casterX, casterY, casterLevel, err := loadCaster(l, ctx, characterId)
			if err != nil {
				l.WithError(err).Errorf("Resurrection: failed to load caster [%d].", characterId)
				return nil
			}

			recipients := selectByVariant(l, ctx, f, characterId, casterX, casterY, e, info.AffectedPartyMemberBitmap(), skill2.Id(info.SkillId()))

			for _, r := range recipients {
				if hpErr := setHP(l, ctx, f, r.Id(), math.MaxUint16); hpErr != nil {
					l.WithError(hpErr).Errorf("Resurrection: SetHP failed for recipient [%d]; skipping warp.", r.Id())
					continue
				}
				if wErr := warpToPosition(l, ctx, f, r.Id(), r.X(), r.Y()); wErr != nil {
					l.WithError(wErr).Errorf("Resurrection: WarpToPosition failed for recipient [%d].", r.Id())
					continue
				}
				l.Debugf("Resurrection: revived [%d] at (%d,%d).", r.Id(), r.X(), r.Y())
			}

			broadcastEffects(l, ctx, wp, f, characterId, casterLevel, info.SkillId(), info.SkillLevel())

			l.Debugf("Resurrection: caster=[%d] skill=[%d] level=[%d] recipients=[%d].",
				characterId, info.SkillId(), info.SkillLevel(), len(recipients))
			return nil
		}
	}
}
