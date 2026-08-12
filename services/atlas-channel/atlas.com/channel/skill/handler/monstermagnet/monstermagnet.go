// Package monstermagnet implements the Hero / Paladin / Dark Knight Monster
// Magnet cast (task-215). The client picks up to the skill's WZ mobCount nearby
// monsters, rolls a per-monster grab result and sends the table on the use-skill
// opcode; the server validates that table, plays the grab effect on every OTHER
// client in the field, wipes each grabbed monster's damage aggro, and hands the
// monster's controller to the caster.
package monstermagnet

import (
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/monster"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	_map "atlas-channel/map"

	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	monstercb "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
)

func init() {
	// Identity, not wire id: registry.go is keyed on skill2.Identity and
	// UseSkill resolves the incoming wire id through
	// constants.For(...).Skill.Resolve before Lookup (task-187). One
	// registration per identity covers every provisioned version.
	//
	// The Handler registry, NOT AttackCastHandler: the magnet arrives on the
	// use-skill opcode rather than an attack packet, and it deals no damage.
	// Registration here also means UseSkill has already charged the WZ mpCon
	// before Apply runs — Apply must not charge it again.
	channelhandler.Register(skill2.HeroMonsterMagnet, Apply)
	channelhandler.Register(skill2.PaladinMonsterMagnet, Apply)
	channelhandler.Register(skill2.DarkKnightMonsterMagnet, Apply)
}

// grabResultSuccess / grabSuccessFlag are the CATCH_MONSTER writer arguments
// for a successful grab. The wire grab result is a BOOLEAN, not an enum: the
// local client computes its ShowCatchEffect selector as `(grabResult == 3)`
// (gms_83 CMob::OnHit @0x668B83 — the three magnet ids at @0x668DB7/DC3/DCA all
// jump to @0x668E14, which does setz al on (arg == 3) then calls
// ShowCatchEffect @0x668E22). So a successful grab maps to 1.
const (
	grabResultSuccess = byte(1)
	grabSuccessFlag   = byte(1)
)

// loadCasterFunc is the caster-load seam tests replace.
var loadCasterFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (character.Model, error) {
	cp := character.NewProcessor(l, ctx)
	return cp.GetById()(characterId)
}

// rectQueryFunc is the mob-selection seam tests replace.
var rectQueryFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]monster.Model, error) {
	return monster.NewProcessor(l, ctx).GetInMapRect(f, x1, y1, x2, y2, limit)
}

// announceCatchFunc is the grab-effect broadcast seam tests replace.
//
// OTHER sessions only, deliberately. The caster's own client already renders
// the effect locally: TryDoingMonsterMagnet calls CMob::AddDamageInfo per
// grabbed mob, which drives CMob::OnHit -> ShowCatchEffect on that client.
// Sending CATCH_MONSTER to the caster too would play the animation twice —
// exactly the double-render task-212 removed from the catch-item path. Remote
// clients never run AddDamageInfo for the magnet, so they DO need the packet.
var announceCatchFunc = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, casterId uint32, monsterId uint32) error {
	return _map.NewProcessor(l, ctx).ForOtherSessionsInMap(
		f, casterId,
		session.Announce(l)(ctx)(wp)(monstercb.CatchMonsterWriter)(
			writer.CatchMonsterBody(monsterId, grabResultSuccess, grabSuccessFlag)),
	)
}

// clearAggroFunc and forceControlFunc are the two monster-command emit seams.
var clearAggroFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, monsterId uint32) error {
	return monster.NewProcessor(l, ctx).ClearAggro(f, monsterId)
}

var forceControlFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, monsterId uint32, characterId uint32) error {
	return monster.NewProcessor(l, ctx).ForceControl(f, monsterId, characterId)
}

// Apply is the registered Monster Magnet handler. It always returns nil: a
// partial failure must never abort the caller's EnableActions unlock.
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
			sid := skill2.Id(info.SkillId())
			slvl := uint32(info.SkillLevel())

			// 1. Drop failed grabs (FR-2.5) and released slots. objectId 0 is a
			//    legitimate wire value for a slot the client released mid-cast.
			claimed := make([]uint32, 0, len(info.MagnetGrabs()))
			for _, g := range info.MagnetGrabs() {
				if !g.Grabbed() || g.ObjectId() == 0 {
					continue
				}
				claimed = append(claimed, g.ObjectId())
			}
			if len(claimed) == 0 {
				l.WithFields(logrus.Fields{
					"caster":           characterId,
					"skill_id":         uint32(sid),
					"skill_level":      slvl,
					"client_mob_count": len(info.MagnetGrabs()),
					"grabbed":          0,
				}).Debug("monster_magnet_summary")
				return nil
			}

			// 2. FR-2.2 — over-cap rejects the WHOLE cast.
			if channelhandler.ExceedsMobCap(l, "monster_magnet_anomaly_over_cap", characterId, sid, slvl, e.MobCount(), claimed) {
				return nil
			}

			// 3. FR-2.7 — caster load failure drops the whole cast.
			c, cErr := loadCasterFunc(l, ctx, characterId)
			if cErr != nil {
				l.WithError(cErr).WithFields(logrus.Fields{
					"event":        "monster_magnet_caster_load_failed",
					"character_id": characterId,
					"skill_id":     uint32(sid),
				}).Error("monster_magnet_caster_load_failed")
				return nil
			}

			// 4. Region check. Exactly ONE rect query per cast regardless of how
			//    many monsters were claimed.
			applied := claimed
			var anomaly []uint32
			rect := [4]int16{}
			if e.Range() > 0 {
				facingLeft := (c.Stance() & 1) == 1
				x1, y1, x2, y2 := channelhandler.MagnetRegion(c.X(), c.Y(), facingLeft, e.Range())
				rect = [4]int16{x1, y1, x2, y2}

				mobs, qErr := rectQueryFunc(l, ctx, f, x1, y1, x2, y2, e.MobCount())
				if qErr != nil {
					l.WithError(qErr).WithFields(logrus.Fields{
						"event":        "monster_magnet_rect_query_failed",
						"character_id": characterId,
						"skill_id":     uint32(sid),
						"rect":         rect,
					}).Error("monster_magnet_rect_query_failed")
					return nil
				}
				serverMobIds := make([]uint32, 0, len(mobs))
				for _, m := range mobs {
					serverMobIds = append(serverMobIds, m.UniqueId())
				}

				applied, anomaly = channelhandler.IntersectMobIds(claimed, serverMobIds)

				// FR-2.3 — an out-of-region target is dropped INDIVIDUALLY.
				if len(anomaly) > 0 {
					l.WithFields(logrus.Fields{
						"event":           "monster_magnet_anomaly_out_of_rect",
						"character_id":    characterId,
						"skill_id":        uint32(sid),
						"skill_level":     slvl,
						"rect":            map[string]int16{"x1": x1, "y1": y1, "x2": x2, "y2": y2},
						"mob_count_cap":   e.MobCount(),
						"client_mob_ids":  claimed,
						"server_mob_ids":  serverMobIds,
						"anomaly_mob_ids": anomaly,
					}).Warn("client_targeted_mob_outside_server_rect")
				}
			} else {
				// FR-2.4's fallback, relocated from lt/rb to `range`: no region
				// contract in this tenant's WZ data, so accept the client's list
				// subject to the cap only. Defensive — `range` is present for all
				// three magnet skills at every level in the data read for design
				// section 3.
				l.WithFields(logrus.Fields{
					"skill_id":         uint32(sid),
					"skill_level":      slvl,
					"client_mob_count": len(claimed),
				}).Debug("monster_magnet_no_range_cap_only")
			}

			// 5. Per surviving monster: grab effect, then aggro wipe, then
			//    controller handover. CLEAR_AGGRO must precede FORCE_CONTROL —
			//    both key on the monster id so they land on the same partition in
			//    this order, and reversing them would have the wipe immediately
			//    clear the aggro flag the handover just set.
			grabbed := 0
			for _, monsterId := range applied {
				if aErr := announceCatchFunc(l, ctx, wp, f, characterId, monsterId); aErr != nil {
					l.WithError(aErr).Warnf("Monster Magnet: unable to broadcast the grab effect for monster [%d].", monsterId)
				}
				if cmErr := clearAggroFunc(l, ctx, f, monsterId); cmErr != nil {
					l.WithError(cmErr).Warnf("Monster Magnet: unable to clear aggro for monster [%d].", monsterId)
				}
				if fErr := forceControlFunc(l, ctx, f, monsterId, characterId); fErr != nil {
					l.WithError(fErr).Warnf("Monster Magnet: unable to force control of monster [%d] to character [%d].", monsterId, characterId)
				}
				grabbed++
			}

			l.WithFields(logrus.Fields{
				"caster":              characterId,
				"skill_id":            uint32(sid),
				"skill_level":         slvl,
				"client_mob_count":    len(info.MagnetGrabs()),
				"claimed":             len(claimed),
				"grabbed":             grabbed,
				"out_of_rect_dropped": len(anomaly),
				"rect":                rect,
			}).Debug("monster_magnet_summary")

			return nil
		}
	}
}
