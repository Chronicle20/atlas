package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	"atlas-channel/compartment"
	"atlas-channel/data/skill/effect"
	compartmentMsg "atlas-channel/kafka/message/compartment"
	once "atlas-channel/kafka/once/compartment"
	"atlas-channel/monster"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventoryconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	itemconst "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

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
				consumeSkillItem(l, ctx, characterId, info.SkillId(), itemId)
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

// consumeSkillItem charges one of itemId from the caster's inventory using
// the compartment Reserve→OneTime→Consume saga (mirrors the projectile
// emission pattern in character_attack_projectile.go). Generic across every
// itemConsume skill (Priest Doom's Magic Rock, summons, Mystic Door, etc.).
//
// Why not consumable.RequestItemConsume: that path starts a saga whose
// completion is gated on a downstream side-effect emit (HP/MP change, scroll
// result) that skill casts don't produce. The reservation would never commit
// and the inventory count would not decrement.
//
// Missing item is logged at WARN and the cast still proceeds (defense-in-depth;
// the v83 client gates the cast UI on item availability).
func consumeSkillItem(l logrus.FieldLogger, ctx context.Context, characterId uint32, skillId uint32, itemId uint32) {
	cp := character.NewProcessor(l, ctx)
	c, cErr := cp.GetById(cp.InventoryDecorator)(characterId)
	if cErr != nil {
		l.WithError(cErr).Warnf("Character [%d] cast skill [%d] requiring item [%d] but failed to load inventory; cast permitted.", characterId, skillId, itemId)
		return
	}
	invType, typeOk := inventoryconst.TypeFromItemId(itemconst.Id(itemId))
	if !typeOk {
		l.Warnf("Character [%d] cast skill [%d] requiring item [%d] but item id has no inventory type; cast permitted.", characterId, skillId, itemId)
		return
	}
	a, found := c.Inventory().CompartmentByType(invType).FindFirstByItemId(itemId)
	if !found {
		l.Warnf("Character [%d] cast skill [%d] requiring item [%d] but no such item found in inventory; cast permitted (defense-in-depth gate only).", characterId, skillId, itemId)
		return
	}

	cpp := compartment.NewProcessor(l, ctx)
	t, tErr := topic.EnvProvider(l)(compartmentMsg.EnvEventTopicStatus)()
	if tErr != nil {
		l.WithError(tErr).Warnf("Character [%d] cast skill [%d] requiring item [%d] but compartment status topic unavailable; cast permitted.", characterId, skillId, itemId)
		return
	}
	txId := uuid.New()
	slot := a.Slot()
	validator := once.ReservationValidator(txId, itemId)
	handler := func(_ logrus.FieldLogger, _ context.Context, _ compartmentMsg.StatusEvent[compartmentMsg.ReservedEventBody]) {
		if cErr := cpp.Consume(txId, characterId, invType, slot); cErr != nil {
			l.WithError(cErr).Errorf("Character [%d] skill [%d] item [%d]: failed to emit CONSUME for reservation [%s].", characterId, skillId, itemId, txId)
		}
	}
	if _, rErr := consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(validator, handler))); rErr != nil {
		l.WithError(rErr).Warnf("Character [%d] cast skill [%d] requiring item [%d] but failed to register one-time consume handler; cast permitted.", characterId, skillId, itemId)
		return
	}
	reserves := []compartmentMsg.ItemBody{{Source: slot, ItemId: itemId, Quantity: 1}}
	if rErr := cpp.RequestReserve(txId, characterId, invType, reserves); rErr != nil {
		l.WithError(rErr).Errorf("Character [%d] cast skill [%d] item [%d]: failed to emit REQUEST_RESERVE.", characterId, skillId, itemId)
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
