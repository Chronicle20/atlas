package mysticdoor

import (
	"context"

	"atlas-channel/character"
	"atlas-channel/character/buff"
	datamap "atlas-channel/data/map"
	"atlas-channel/data/skill/effect"
	"atlas-channel/data/skill/effect/statup"
	"atlas-channel/door"
	"atlas-channel/socket/writer"
	channelhandler "atlas-channel/skill/handler"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

func init() {
	channelhandler.Register(skill2.PriestMysticDoorId, Apply)
}

// loadMap retrieves the map's fieldLimit, town flag, and whether a valid
// return map exists. hasReturn is derived from ReturnMapId being set to
// something other than EmptyMapId. The atlas-doors engine does an
// authoritative forced+return re-check; this is a cheap channel-side gate.
var loadMap = func(l logrus.FieldLogger, ctx context.Context, mapId _map.Id) (fieldLimit uint32, town bool, hasReturn bool, err error) {
	m, err := datamap.NewProcessor(l, ctx).GetById(mapId)
	if err != nil {
		return 0, false, false, err
	}
	return m.FieldLimit(), m.Town(), m.ReturnMapId() != _map.EmptyMapId, nil
}

// loadCaster returns the caster's (X, Y) position from the character service.
var loadCaster = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (int16, int16, error) {
	c, err := character.NewProcessor(l, ctx).GetById()(characterId)
	if err != nil {
		return 0, 0, err
	}
	return c.X(), c.Y(), nil
}

// emitSpawn sends the SPAWN command to atlas-doors via the G1 door processor.
var emitSpawn = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId, skillId uint32, level byte, x, y int16) error {
	return door.NewProcessor(l, ctx).Spawn(f, characterId, skillId, level, x, y)
}

// applyDoorBuff gives the caster the Mystic Door buff so the client shows the
// duration icon (and lets the player right-click it to cancel, which atlas turns
// into an early door removal — see socket/handler/character_buff_cancel.go).
//
// Mystic Door's WZ effect carries no statups, so the generic skill-use buff apply
// skips it (it gates on len(StatUps) > 0) and nothing would render. The v83 client
// only draws a buff icon for a SET character-temporary-stat bit, and the cast
// itself (CUserLocal::DoActiveSkill_OpenGate @0x969bf9) sets none — so a stat-less
// buff is invisible. Following the v83 reference server, the door reuses the
// SOUL_ARROW temporary stat (gameplay-harmless for a Priest, who carries no bow)
// with the skill's X value: the buff bar then shows the Mystic Door icon (keyed by
// sourceId 2311002) and is cancellable. The buff's duration mirrors the door's
// lifetime, so it expires alongside the door. Stubbable for tests.
var applyDoorBuff = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId, skillId uint32, level byte, duration int32, amount int16) {
	if duration <= 0 {
		return
	}
	statups := []statup.Model{statup.NewModel(string(charconst.TemporaryStatTypeSoulArrow), int32(amount))}
	_ = buff.NewProcessor(l, ctx).Apply(f, characterId, int32(skillId), level, duration, statups)(characterId)
}

// Apply is the Mystic Door handler installed in the per-skill registry.
//
// By the time this handler runs, UseSkill has already consumed MP + Magic Rock
// and skipped the character buff (Mystic Door has no statups). This handler
// performs cheap channel-side eligibility rejections (field limit, town map,
// no return map) and, if eligible, emits a SPAWN command with the caster's
// current position. Rejections emit nothing — the client was already
// re-enabled by UseSkill.
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
			wp writer.Producer,
			f field.Model,
			characterId uint32,
			info packetmodel.SkillUsageInfo,
			e effect.Model,
		) error {
			fieldLimit, town, hasReturn, err := loadMap(l, ctx, f.MapId())
			if err != nil {
				l.WithError(err).Warnf("Mystic Door: map lookup failed for map [%d].", f.MapId())
				return nil
			}
			if town || !hasReturn || fieldLimit&_map.FieldLimitNoMysticDoor != 0 {
				l.Debugf("Mystic Door: rejected cast by [%d] — town=%v hasReturn=%v limit=0x%x.",
					characterId, town, hasReturn, fieldLimit)
				return nil
			}

			x, y, err := loadCaster(l, ctx, characterId)
			if err != nil {
				l.WithError(err).Errorf("Mystic Door: failed to load caster [%d].", characterId)
				return nil
			}

			if err := emitSpawn(l, ctx, f, characterId, uint32(info.SkillId()), info.SkillLevel(), x, y); err != nil {
				return err
			}

			// Give the caster the Mystic Door buff (duration icon + right-click cancel).
			applyDoorBuff(l, ctx, f, characterId, uint32(info.SkillId()), info.SkillLevel(), e.Duration(), e.X())
			return nil
		}
	}
}
