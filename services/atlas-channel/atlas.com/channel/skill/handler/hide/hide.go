package hide

import (
	"context"
	"math"

	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/data/skill/effect"
	"atlas-channel/data/skill/effect/statup"
	_mapconsumer "atlas-channel/kafka/consumer/map"
	"atlas-channel/session"
	channelhandler "atlas-channel/skill/handler"
	socketHandler "atlas-channel/socket/handler"
	"atlas-channel/socket/writer"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

// HideBuffDuration is the effectively-permanent duration for the GM-hide buff.
// atlas-buffs rejects duration <= 0, so the toggle uses the largest int32
// (~24.8 days); the canonical reveal is a re-cast, exactly like mounts.
const HideBuffDuration = int32(math.MaxInt32)

func init() {
	channelhandler.Register(skill2.SuperGmHideId, Apply)
}

// hideDeps holds the Hide toggle's collaborators as function seams so the
// direction logic is unit-testable offline. announceSelf takes the caster level
// so the wiring builds the skill-use packet without re-loading the caster.
// There is deliberately NO foreign-announce seam: the Hide skill never
// broadcasts a foreign skill-use animation (it would leak GM presence in both
// toggle directions — see task-156 plan Global Constraints).
type hideDeps struct {
	loadCaster        func(characterId uint32) (character.Model, error)
	isHidden          func(characterId uint32) (bool, error)
	applyHide         func(f field.Model, characterId uint32, level byte) error
	cancelHide        func(f field.Model, characterId uint32) error
	despawnFromOthers func(f field.Model, characterId uint32) error
	spawnToOthers     func(f field.Model, characterId uint32) error
	announceSelf      func(level byte) error
}

// applyHide is the tested core: gate SuperGM, read current hide state, then
// toggle. Hide ON applies the buff and despawns the caster from others; hide
// OFF cancels the buff and spawns the caster back. Self animation always fires;
// no foreign animation is broadcast.
func applyHide(l logrus.FieldLogger, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, d hideDeps) error {
	c, err := d.loadCaster(characterId)
	if err != nil {
		l.WithError(err).Errorf("Hide: failed to load caster [%d].", characterId)
		return nil
	}
	if !job.IsA(c.JobId(), job.SuperGmId) {
		l.Warnf("Character [%d] cast SuperGM Hide without SuperGM job; rejecting.", characterId)
		return nil
	}

	hidden, hErr := d.isHidden(characterId)
	if hErr != nil {
		l.WithError(hErr).Debugf("Hide: unable to resolve hide state for caster [%d]; treating as visible.", characterId)
		hidden = false
	}

	if !hidden {
		// Hide ON.
		if err := d.applyHide(f, characterId, info.SkillLevel()); err != nil {
			l.WithError(err).Errorf("Hide: failed to apply hide buff for caster [%d].", characterId)
		}
		if err := d.despawnFromOthers(f, characterId); err != nil {
			l.WithError(err).Errorf("Hide: failed to despawn caster [%d] from others.", characterId)
		}
	} else {
		// Hide OFF (reveal).
		if err := d.cancelHide(f, characterId); err != nil {
			l.WithError(err).Errorf("Hide: failed to cancel hide buff for caster [%d].", characterId)
		}
		if err := d.spawnToOthers(f, characterId); err != nil {
			l.WithError(err).Errorf("Hide: failed to spawn caster [%d] to others.", characterId)
		}
	}

	if err := d.announceSelf(c.Level()); err != nil {
		l.WithError(err).Debugf("Hide: self skill-use announce failed for caster [%d].", characterId)
	}
	return nil
}

// Apply is the registered Hide handler. It builds production deps and delegates
// to applyHide.
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
			cp := character.NewProcessor(l, ctx)
			bp := buff.NewProcessor(l, ctx)
			sp := session.NewProcessor(l, ctx)

			d := hideDeps{
				loadCaster: func(id uint32) (character.Model, error) { return cp.GetById()(id) },
				isHidden: func(id uint32) (bool, error) {
					bs, err := bp.GetByCharacterId(id)
					if err != nil {
						return false, err
					}
					return buff.IsGmHidden(bs), nil
				},
				applyHide: func(f field.Model, id uint32, level byte) error {
					// DARK_SIGHT amount must be non-zero: the v83 client's
					// CUser::IsDarkSight tests the stat != 0.
					statups := []statup.Model{statup.NewModel(string(charconst.TemporaryStatTypeDarkSight), 1)}
					return bp.Apply(f, id, int32(skill2.SuperGmHideId), level, HideBuffDuration, statups)(id)
				},
				cancelHide: func(f field.Model, id uint32) error {
					return bp.Cancel(f, id, int32(skill2.SuperGmHideId))
				},
				despawnFromOthers: func(f field.Model, id uint32) error {
					return _mapconsumer.DespawnCharacterInMap(l, ctx, wp)(f, id)
				},
				spawnToOthers: func(f field.Model, id uint32) error {
					return _mapconsumer.SpawnCharacterInMap(l, ctx, wp)(f, id)
				},
				announceSelf: func(level byte) error {
					return sp.IfPresentByCharacterId(f.Channel())(
						characterId,
						socketHandler.AnnounceSkillUse(l)(ctx)(wp)(info.SkillId(), level, info.SkillLevel()),
					)
				},
			}
			return applyHide(l, f, characterId, info, d)
		}
	}
}
