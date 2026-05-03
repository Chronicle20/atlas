package doom

import (
	"context"
	"math/rand"

	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/monster"
	channelhandler "atlas-channel/skill/handler"
	"atlas-channel/socket/writer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func init() {
	channelhandler.Register(skill2.PriestDoomId, Apply)
}

// loadCasterFunc is the caster-load seam tests can replace. Production
// calls atlas-character via character.NewProcessor(...).GetById(); tests
// inject a stub returning a deterministic character.Model so the handler
// can exercise its mob-selection / status-apply logic offline.
var loadCasterFunc = func(cp character.Processor, characterId uint32) (character.Model, error) {
	return cp.GetById()(characterId)
}

// propRollFunc gates per-target apply by the skill's prop value. Production
// uses a uniform RNG; tests inject a deterministic implementation via a
// t.Cleanup-restored override.
var propRollFunc = func(prop float64) bool {
	if prop <= 0 {
		return false
	}
	if prop >= 1 {
		return true
	}
	return rand.Float64() <= prop
}

// rectQueryFunc is the mob-selection seam tests can replace. Production
// calls atlas-monsters via monster.NewProcessor(...).GetInMapRect; tests
// inject a stub returning a fixed slice.
var rectQueryFunc = func(p *monster.Processor, f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]monster.Model, error) {
	return p.GetInMapRect(f, x1, y1, x2, y2, limit)
}

// applyStatusFunc is the status-emit seam tests can replace.
var applyStatusFunc = func(p *monster.Processor, f field.Model, monsterId, characterId, skillId, skillLevel uint32, statuses map[string]int32, duration uint32) error {
	return p.ApplyStatus(f, monsterId, characterId, skillId, skillLevel, statuses, duration)
}

// reflectLookupFunc is the magic-reflect probe seam tests can replace.
var reflectLookupFunc = func(t tenant.Model, monsterId uint32, kind string) (monster.ReflectInfo, bool) {
	return monster.GetStatusMirror().GetReflect(t, monsterId, kind)
}

// Apply is the Priest Doom handler installed in the per-skill registry.
//
// Lifecycle (mirrors Cosmic StatEffect.applyMonsterBuff):
//  1. Load caster (X, Y, Stance).
//  2. Derive isFacingLeft from Stance (odd = facing left, OdinMS convention).
//  3. Compute the cast's bounding box from caster pos + facing + e.LT()/e.RB().
//  4. Query atlas-monsters for monsters in that rectangle, capped at e.MobCount().
//  5. Per mob: skip on active magic-reflect; skip on prop RNG miss; otherwise
//     emit ApplyStatus({DOOM:1}) for the skill's duration.
//  6. Emit a per-cast summary log line.
//
// HP/MP/itemConsume/cooldown costs are charged by handler.UseSkill before
// this handler runs (common.go cost block). This handler is responsible
// only for mob selection + per-mob apply.
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer,
	f field.Model, characterId uint32,
	info packetmodel.SkillUsageInfo, e effect.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer,
		f field.Model, characterId uint32,
		info packetmodel.SkillUsageInfo, e effect.Model,
	) error {
		return func(
			wp writer.Producer,
			f field.Model, characterId uint32,
			info packetmodel.SkillUsageInfo, e effect.Model,
		) error {
			cp := character.NewProcessor(l, ctx)
			c, err := loadCasterFunc(cp, characterId)
			if err != nil {
				l.WithError(err).Errorf("Doom: failed to load caster [%d].", characterId)
				return nil
			}

			facingLeft := (c.Stance() & 1) == 1
			x1, y1, x2, y2 := calculateBoundingBox(c.X(), c.Y(), facingLeft, e.LT(), e.RB())

			mp := monster.NewProcessor(l, ctx)
			mobs, qErr := rectQueryFunc(mp, f, x1, y1, x2, y2, e.MobCount())
			if qErr != nil {
				l.WithError(qErr).Errorf("Doom: rect query failed for caster [%d].", characterId)
				return nil
			}

			t := tenant.MustFromContext(ctx)
			statuses := map[string]int32{monster2.StatusDoom: 1}
			applied, reflectSkipped, propSkipped := 0, 0, 0
			for _, m := range mobs {
				if _, ok := reflectLookupFunc(t, m.UniqueId(), monster2.ReflectKindMagical); ok {
					l.Debugf("Doom: monster [%d] has MAGICAL reflect; status apply skipped.", m.UniqueId())
					reflectSkipped++
					continue
				}
				if !propRollFunc(e.Prop()) {
					propSkipped++
					continue
				}
				_ = applyStatusFunc(mp, f, m.UniqueId(), characterId, uint32(skill2.PriestDoomId), uint32(info.SkillLevel()), statuses, uint32(e.Duration()))
				applied++
			}

			l.Debugf("Doom: caster=[%d] level=[%d] mobsInRect=[%d] applied=[%d] reflectSkipped=[%d] propSkipped=[%d].",
				characterId, info.SkillLevel(), len(mobs), applied, reflectSkipped, propSkipped)
			return nil
		}
	}
}
