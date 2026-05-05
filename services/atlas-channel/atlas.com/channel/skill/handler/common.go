package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	"atlas-channel/consumable"
	"atlas-channel/data/skill/effect"
	"atlas-channel/monster"
	"atlas-channel/socket/writer"
	"context"
	"math/rand"

	charcon "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventoryconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	itemconst "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// loadCasterFunc is the caster-load seam tests can replace. Production
// calls atlas-character via character.Processor.GetById(); tests inject a
// stub returning a deterministic character.Model so the orchestrator can
// exercise its mob-selection / status-apply logic offline.
var loadCasterFunc = func(cp character.Processor, characterId uint32) (character.Model, error) {
	return cp.GetById()(characterId)
}

// rectQueryFunc is the mob-selection seam tests can replace. Production
// calls atlas-monsters via monster.Processor.GetInMapRect; tests inject a
// stub returning a fixed slice.
var rectQueryFunc = func(p *monster.Processor, f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]monster.Model, error) {
	return p.GetInMapRect(f, x1, y1, x2, y2, limit)
}

// propRollFunc gates per-target apply/cancel by the skill's prop value.
// Production uses a uniform RNG; tests inject a deterministic implementation
// via a t.Cleanup-restored override.
var propRollFunc = func(prop float64) bool {
	if prop <= 0 {
		return false
	}
	if prop >= 1 {
		return true
	}
	return rand.Float64() <= prop
}

// reflectLookupFunc is the magic-reflect probe seam tests can replace.
var reflectLookupFunc = func(t tenant.Model, monsterId uint32, kind string) (monster.ReflectInfo, bool) {
	return monster.GetStatusMirror().GetReflect(t, monsterId, kind)
}

// applyStatusFunc is the status-apply emit seam tests can replace.
var applyStatusFunc = func(p *monster.Processor, f field.Model, monsterId, characterId, skillId, skillLevel uint32, statuses map[string]int32, duration uint32) error {
	return p.ApplyStatus(f, monsterId, characterId, skillId, skillLevel, statuses, duration)
}

// cancelStatusFunc is the status-cancel emit seam tests can replace.
var cancelStatusFunc = func(p *monster.Processor, f field.Model, monsterId uint32, statusTypes []string, sourceCharacterId, sourceSkillId uint32, sourceSkillClass string) error {
	return p.CancelStatus(f, monsterId, statusTypes, sourceCharacterId, sourceSkillId, sourceSkillClass)
}

func UseSkill(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) error {
	return func(ctx context.Context) func(wp writer.Producer, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) error {
		return func(wp writer.Producer, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) error {
			if e.HPConsume() > 0 {
				_ = character.NewProcessor(l, ctx).ChangeHP(f, characterId, -int16(e.HPConsume()))
			}
			if e.MPConsume() > 0 {
				_ = character.NewProcessor(l, ctx).ChangeMP(f, characterId, -int16(e.MPConsume()))
			}
			if itemId := e.ItemConsume(); itemId > 0 {
				cp := character.NewProcessor(l, ctx)
				if c, cErr := cp.GetById(cp.InventoryDecorator)(characterId); cErr == nil {
					if invType, typeOk := inventoryconst.TypeFromItemId(itemconst.Id(itemId)); typeOk {
						if a, found := c.Inventory().CompartmentByType(invType).FindFirstByItemId(itemId); found {
							_ = consumable.NewProcessor(l, ctx).RequestItemConsume(f, charcon.Id(characterId), itemconst.Id(itemId), slot.Position(a.Slot()), 0)
						} else {
							l.Warnf("Character [%d] cast skill [%d] requiring item [%d] but no such item found in inventory; cast permitted (defense-in-depth gate only).", characterId, info.SkillId(), itemId)
						}
					}
				} else {
					l.WithError(cErr).Warnf("Character [%d] cast skill [%d] requiring item [%d] but failed to load inventory; cast permitted.", characterId, info.SkillId(), itemId)
				}
			}
			if e.Cooldown() > 0 {
				_ = skill.NewProcessor(l, ctx).ApplyCooldown(f, skill2.Id(info.SkillId()), e.Cooldown())(characterId)
			}
			if e.Duration() > 0 && len(e.StatUps()) > 0 {
				applyBuffFunc := buff.NewProcessor(l, ctx).Apply(f, characterId, int32(info.SkillId()), info.SkillLevel(), e.Duration(), e.StatUps())
				_ = applyBuffFunc(characterId)
				casterX, casterY := int16(0), int16(0)
				if c, cErr := character.NewProcessor(l, ctx).GetById()(characterId); cErr == nil {
					casterX, casterY = c.X(), c.Y()
				}
				_ = applyToParty(l)(ctx)(f, characterId, casterX, casterY, e, info.AffectedPartyMemberBitmap())(applyBuffFunc)
			}

			// Handle mob-affecting buffs (crash, dispel, etc.)
			applyToMobs(l, ctx, f, characterId, info, e)

			// Per-skill dispatcher (Heal, Dispel, Cure, MPEater, Drain, ...).
			if h, ok := Lookup(skill2.Id(info.SkillId())); ok {
				if err := h(l)(ctx)(wp, f, characterId, info, e); err != nil {
					l.WithError(err).Errorf("Skill handler for [%d] failed for character [%d].", info.SkillId(), characterId)
				}
			}

			return nil
		}
	}
}

func applyToMobs(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) {
	mobIds := info.AffectedMobIds()
	if len(mobIds) == 0 {
		return
	}

	sid := skill2.Id(info.SkillId())
	slvl := uint32(info.SkillLevel())
	mobCap := e.MobCount()

	// FR-4.3 — mobCount cap. Reject the entire cast if the client claims more
	// targets than the skill's WZ definition permits. This runs before any
	// atlas-monsters round-trip; an over-cap cast produces zero emit calls.
	if uint32(len(mobIds)) > mobCap {
		l.WithFields(logrus.Fields{
			"event":            "monster_buff_anomaly_over_cap",
			"character_id":     characterId,
			"skill_id":         uint32(sid),
			"skill_level":      slvl,
			"mob_count_cap":    mobCap,
			"client_mob_count": len(mobIds),
			"client_mob_ids":   mobIds,
		}).Warn("client_target_count_exceeds_skill_cap")
		return
	}

	mp := monster.NewProcessor(l, ctx)

	var (
		applied         []uint32
		anomaly         []uint32
		mobsInRectCount = -1
		rect            [4]int16 // x1, y1, x2, y2 — only meaningful when bbox present
	)

	if !hasEffectBbox(e.LT(), e.RB()) {
		// FR-4.2 — no rect contract in WZ data; trust the client unmodified
		// for the rect check. Cap (already done), prop, reflect still apply.
		l.WithFields(logrus.Fields{
			"skill_id":         uint32(sid),
			"skill_level":      slvl,
			"client_mob_count": len(mobIds),
		}).Debug("mob_buff_no_effect_bbox")
		applied = mobIds
	} else {
		// FR-4.1 — rect verification. Bail-on-error policy: any failure
		// drops the cast. See design §5.1.
		cp := character.NewProcessor(l, ctx)
		c, cErr := loadCasterFunc(cp, characterId)
		if cErr != nil {
			l.WithError(cErr).WithFields(logrus.Fields{
				"event":        "mob_buff_caster_load_failed",
				"character_id": characterId,
				"skill_id":     uint32(sid),
			}).Error("mob_buff_caster_load_failed")
			return
		}
		facingLeft := (c.Stance() & 1) == 1
		x1, y1, x2, y2 := calculateBoundingBox(c.X(), c.Y(), facingLeft, e.LT(), e.RB())
		rect = [4]int16{x1, y1, x2, y2}

		mobs, qErr := rectQueryFunc(mp, f, x1, y1, x2, y2, mobCap)
		if qErr != nil {
			l.WithError(qErr).WithFields(logrus.Fields{
				"event":        "mob_buff_rect_query_failed",
				"character_id": characterId,
				"skill_id":     uint32(sid),
				"rect":         rect,
			}).Error("mob_buff_rect_query_failed")
			return
		}
		serverMobIds := make([]uint32, 0, len(mobs))
		for _, m := range mobs {
			serverMobIds = append(serverMobIds, m.UniqueId())
		}
		mobsInRectCount = len(serverMobIds)

		applied, anomaly = intersectMobIds(mobIds, serverMobIds)

		if len(anomaly) > 0 {
			l.WithFields(logrus.Fields{
				"event":           "monster_buff_anomaly_out_of_rect",
				"character_id":    characterId,
				"skill_id":        uint32(sid),
				"skill_level":     slvl,
				"rect":            map[string]int16{"x1": x1, "y1": y1, "x2": x2, "y2": y2},
				"mob_count_cap":   mobCap,
				"client_mob_ids":  mobIds,
				"server_mob_ids":  serverMobIds,
				"anomaly_mob_ids": anomaly,
			}).Warn("client_targeted_mob_outside_server_rect")
		}
	}

	t := tenant.MustFromContext(ctx)
	monsterStatuses := make(map[string]int32, len(e.MonsterStatus()))
	for k, v := range e.MonsterStatus() {
		monsterStatuses[k] = int32(v)
	}

	isCancel := isCrashOrDispel(sid)
	cancelClass := ""
	if isCancel {
		cancelClass = dispelSkillClass(sid)
	}

	// Branch selection mirrors the FR-4.9 rule: a skill takes EITHER the
	// cancel branch (Crash family / Priest Dispel) OR the apply branch
	// (Doom and any future entry with non-empty MonsterStatus). Never both.
	branch := propBranchApply
	if isCancel {
		branch = propBranchCancel
	} else if len(monsterStatuses) == 0 {
		// Buff-classified skill with no MonsterStatus map — defensive: nothing
		// to apply. Should not occur for skills in isMobAffectingBuff today.
		l.WithFields(logrus.Fields{
			"skill_id": uint32(sid),
		}).Debug("mob_buff_no_emit_branch")
		l.WithFields(buildSummaryFields(characterId, sid, slvl, mobsInRectCount, len(mobIds), 0, 0, 0, len(anomaly))).Debug("mob_buff_apply_summary")
		return
	}

	appliedCount, reflectSkipped, propSkipped := 0, 0, 0
	for _, mobId := range applied {
		// FR-4.6 — kind-aware reflect skip.
		var kind string
		if isCancel {
			kind = cancelClass
		} else {
			kind = mobBuffApplyKind(sid)
		}
		if kind == "" {
			l.WithFields(logrus.Fields{
				"event":    "mob_buff_unclassified_kind",
				"skill_id": uint32(sid),
				"mob_id":   mobId,
			}).Debug("mob_buff_unclassified_kind")
		} else if _, hasReflect := reflectLookupFunc(t, mobId, kind); hasReflect {
			l.WithFields(logrus.Fields{
				"skill_id": uint32(sid),
				"mob_id":   mobId,
				"kind":     kind,
			}).Debug("mob_buff_reflect_skip")
			reflectSkipped++
			continue
		}

		// FR-4.5 — prop roll, with per-skill carve-out support.
		if propAppliesTo(sid, branch) {
			if !propRollFunc(e.Prop()) {
				propSkipped++
				continue
			}
		}

		// FR-4.9 — branch emit.
		if isCancel {
			_ = cancelStatusFunc(mp, f, mobId, nil, characterId, uint32(sid), cancelClass)
		} else {
			_ = applyStatusFunc(mp, f, mobId, characterId, uint32(sid), slvl, monsterStatuses, uint32(e.Duration()))
		}
		appliedCount++
	}

	l.WithFields(buildSummaryFields(characterId, sid, slvl, mobsInRectCount, len(mobIds), appliedCount, reflectSkipped, propSkipped, len(anomaly))).Debug("mob_buff_apply_summary")
}

// buildSummaryFields packs the FR-4.8 per-cast summary fields.
func buildSummaryFields(characterId uint32, sid skill2.Id, slvl uint32, mobsInRect, clientMobCount, applied, reflectSkipped, propSkipped, outOfRectDropped int) logrus.Fields {
	return logrus.Fields{
		"caster":              characterId,
		"skill_id":            uint32(sid),
		"skill_level":         slvl,
		"mobs_in_rect":        mobsInRect,
		"client_mob_count":    clientMobCount,
		"applied":             applied,
		"reflect_skipped":     reflectSkipped,
		"prop_skipped":        propSkipped,
		"out_of_rect_dropped": outOfRectDropped,
	}
}

func isCrashOrDispel(skillId skill2.Id) bool {
	return skill2.Is(skillId,
		skill2.CrusaderArmorCrashId,
		skill2.WhiteKnightMagicCrashId,
		skill2.DragonKnightPowerCrashId,
		skill2.PriestDispelId,
	)
}

// dispelSkillClass classifies a crash/dispel skill by the attacker's hit
// class — warrior crashes are physical melee, Priest Dispel is magic. The
// returned string matches atlas-monsters' monster.ReflectKind* constants.
// Returns "" for unrecognized skills so the downstream guard falls through
// to normal cancel semantics.
func dispelSkillClass(skillId skill2.Id) string {
	switch {
	case skill2.Is(skillId,
		skill2.CrusaderArmorCrashId,
		skill2.WhiteKnightMagicCrashId,
		skill2.DragonKnightPowerCrashId):
		return monster2.ReflectKindPhysical
	case skill2.Is(skillId, skill2.PriestDispelId):
		return monster2.ReflectKindMagical
	default:
		return ""
	}
}

func applyToParty(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, casterId uint32, casterX, casterY int16, e effect.Model, memberBitmap byte) func(idOperator model2.Operator[uint32]) error {
	return func(ctx context.Context) func(f field.Model, casterId uint32, casterX, casterY int16, e effect.Model, memberBitmap byte) func(idOperator model2.Operator[uint32]) error {
		return func(f field.Model, casterId uint32, casterX, casterY int16, e effect.Model, memberBitmap byte) func(idOperator model2.Operator[uint32]) error {
			return func(idOperator model2.Operator[uint32]) error {
				recipients := SelectInRangePartyMembers(l, ctx, f, casterId, casterX, casterY, e, memberBitmap)
				for _, r := range recipients {
					_ = idOperator(r.Id())
				}
				return nil
			}
		}
	}
}
