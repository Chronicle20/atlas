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

	mp := monster.NewProcessor(l, ctx)
	sid := skill2.Id(info.SkillId())

	// Crash and Priest Dispel cancel monster self-buffs. We classify the
	// originating skill so atlas-monsters can refuse a same-kind dispel
	// against an active reflect (FR-4.9.1.2).
	if isCrashOrDispel(sid) {
		class := dispelSkillClass(sid)
		for _, mobId := range mobIds {
			_ = mp.CancelStatus(f, mobId, nil, characterId, uint32(info.SkillId()), class)
		}
	}

	// Apply monster status effects from skill (e.g., crash debuff)
	if len(e.MonsterStatus()) > 0 {
		ms := make(map[string]int32)
		for k, v := range e.MonsterStatus() {
			ms[k] = int32(v)
		}
		for _, mobId := range mobIds {
			_ = mp.ApplyStatus(f, mobId, characterId, uint32(info.SkillId()), uint32(info.SkillLevel()), ms, uint32(e.Duration()))
		}
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
// returned string matches atlas-monsters' monster.ReflectKind* constants
// ("PHYSICAL" / "MAGICAL"). Returns "" for unrecognized skills so the
// downstream guard falls through to normal cancel semantics.
func dispelSkillClass(skillId skill2.Id) string {
	switch {
	case skill2.Is(skillId,
		skill2.CrusaderArmorCrashId,
		skill2.WhiteKnightMagicCrashId,
		skill2.DragonKnightPowerCrashId):
		return "PHYSICAL"
	case skill2.Is(skillId, skill2.PriestDispelId):
		return "MAGICAL"
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
