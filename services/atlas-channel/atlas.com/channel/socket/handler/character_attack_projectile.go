package handler

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/compartment"
	"atlas-channel/data/skill/effect"
	compartmentMsg "atlas-channel/kafka/message/compartment"
	once "atlas-channel/kafka/once/compartment"
	"context"
	"sort"

	ts "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	skillConst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type ProjectileSlotDraw struct {
	Slot     int16
	ItemId   uint32
	Quantity int16
}

type ProjectilePlan struct {
	InventoryType inventory.Type
	Draws         []ProjectileSlotDraw
	Required      int
	Available     int
}

func (p *ProjectilePlan) Shortfall() bool {
	return p.Available < p.Required
}

type ProjectileProcessor interface {
	Plan(c character.Model, ai packetmodel.AttackInfo, se effect.Model) (*ProjectilePlan, bool)
	Emit(characterId uint32, plan *ProjectilePlan) error
}

type ProjectileProcessorImpl struct {
	l    logrus.FieldLogger
	ctx  context.Context
	bp   buff.Processor
	cpp  compartment.Processor
}

func NewProjectileProcessor(l logrus.FieldLogger, ctx context.Context) ProjectileProcessor {
	return &ProjectileProcessorImpl{
		l:   l,
		ctx: ctx,
		bp:  buff.NewProcessor(l, ctx),
		cpp: compartment.NewProcessor(l, ctx),
	}
}

func (p *ProjectileProcessorImpl) Plan(c character.Model, ai packetmodel.AttackInfo, se effect.Model) (*ProjectilePlan, bool) {
	if ai.AttackType() != packetmodel.AttackTypeRanged {
		return nil, false
	}
	// TODO(task-007): the javelin packet flag at libs/atlas-packet/model/attack_info.go:153
	// is tied to a specific skill mechanic whose semantics are not yet fully understood
	// (the original name is a poor translation). Consumption is currently skipped when
	// javlin=true to avoid mis-consuming. Revisit once the mechanic is characterized.
	if ai.Javlin() {
		p.l.WithField("characterId", c.Id()).WithField("skillId", ai.SkillId()).
			Debugf("Skipping projectile consumption: javlin flag set.")
		return nil, false
	}
	if skillConst.IsShootSkillNotConsumingBullet(skillConst.Id(ai.SkillId())) {
		p.l.WithField("characterId", c.Id()).WithField("skillId", ai.SkillId()).
			Debugf("Skipping projectile consumption: skill is flagged non-consuming.")
		return nil, false
	}

	weapon, ok := equippedWeapon(c)
	if !ok {
		p.l.WithField("characterId", c.Id()).
			Debugf("Skipping projectile consumption: no weapon equipped.")
		return nil, false
	}
	weaponType := item.GetWeaponType(item.Id(weapon.TemplateId()))
	classification, rangedWeapon := requiredClassification(weaponType)
	if !rangedWeapon {
		p.l.WithField("characterId", c.Id()).WithField("weaponItemId", weapon.TemplateId()).
			Debugf("Skipping projectile consumption: non-ranged weapon.")
		return nil, false
	}

	buffs, err := p.bp.GetByCharacterId(c.Id())
	if err != nil {
		// Treat a buff-lookup failure as "no buffs" so consumption still fires.
		// Soul Arrow is a gameplay-critical skip but we'd rather over-consume than
		// nil-ref the attack hot path.
		p.l.WithError(err).WithField("characterId", c.Id()).
			Warnf("Unable to load buffs for projectile gate; assuming none active.")
		buffs = nil
	}

	if (weaponType == item.WeaponTypeBow || weaponType == item.WeaponTypeCrossbow) && hasBuff(buffs, ts.TemporaryStatTypeSoulArrow) {
		p.l.WithField("characterId", c.Id()).WithField("skillId", ai.SkillId()).
			Debugf("Skipping projectile consumption: Soul Arrow active.")
		return nil, false
	}

	count := computeCount(weaponType, se, buffs)
	if count <= 0 {
		return nil, false
	}

	// TODO(task-007): passive no-consume mechanics (Mortal Blow, Expert Marksmanship,
	// Claw Mastery roll-to-preserve, etc.) are out of scope for v1. These require
	// reading passive skill levels and performing an RNG roll against a per-skill
	// probability to skip the consume. When added, apply before the resolvePlan call.

	draws, available := resolvePlan(c.Inventory().Consumable().Assets(), classification, int16(ai.ProperBulletPosition()), count)
	plan := &ProjectilePlan{
		InventoryType: inventory.TypeValueUse,
		Draws:         draws,
		Required:      count,
		Available:     available,
	}
	if len(plan.Draws) == 0 {
		p.l.WithField("characterId", c.Id()).
			WithField("weaponItemId", weapon.TemplateId()).
			WithField("skillId", ai.SkillId()).
			WithField("required", count).
			WithField("available", available).
			Warnf("No projectile found in inventory for ranged attack.")
		return plan, false
	}
	if plan.Shortfall() {
		p.l.WithField("characterId", c.Id()).
			WithField("weaponItemId", weapon.TemplateId()).
			WithField("skillId", ai.SkillId()).
			WithField("required", count).
			WithField("available", available).
			Warnf("Projectile shortfall on ranged attack; consuming what's available.")
	}
	return plan, true
}

func (p *ProjectileProcessorImpl) Emit(characterId uint32, plan *ProjectilePlan) error {
	if plan == nil || len(plan.Draws) == 0 {
		return nil
	}
	t, err := topic.EnvProvider(p.l)(compartmentMsg.EnvEventTopicStatus)()
	if err != nil {
		return err
	}
	for _, draw := range plan.Draws {
		txId := uuid.New()
		draw := draw
		validator := once.ReservationValidator(txId, draw.ItemId)
		handler := reservedToConsume(p.l, p.cpp, characterId, txId, plan.InventoryType, draw.Slot)
		if _, rerr := consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(validator, handler))); rerr != nil {
			p.l.WithError(rerr).WithField("characterId", characterId).
				Errorf("Unable to register one-time consume handler for projectile reservation.")
			continue
		}
		reserves := []compartmentMsg.ItemBody{{Source: draw.Slot, ItemId: draw.ItemId, Quantity: draw.Quantity}}
		if rerr := p.cpp.RequestReserve(txId, characterId, plan.InventoryType, reserves); rerr != nil {
			p.l.WithError(rerr).WithField("characterId", characterId).WithField("slot", draw.Slot).
				Errorf("Unable to emit projectile reservation request.")
		}
	}
	return nil
}

func reservedToConsume(l logrus.FieldLogger, cpp compartment.Processor, characterId uint32, txId uuid.UUID, invType inventory.Type, slot int16) message.Handler[compartmentMsg.StatusEvent[compartmentMsg.ReservedEventBody]] {
	return func(_ logrus.FieldLogger, _ context.Context, _ compartmentMsg.StatusEvent[compartmentMsg.ReservedEventBody]) {
		if err := cpp.Consume(txId, characterId, invType, slot); err != nil {
			l.WithError(err).WithField("characterId", characterId).WithField("slot", slot).
				Errorf("Unable to emit projectile consume command.")
		}
	}
}

// equippedWeapon returns the asset in the character's main weapon slot (-11) if present.
func equippedWeapon(c character.Model) (asset.Model, bool) {
	s, ok := c.Equipment().Get("weapon")
	if !ok || s.Equipable == nil {
		return asset.Model{}, false
	}
	return *s.Equipable, true
}

func hasBuff(buffs []buff.Model, statType ts.TemporaryStatType) bool {
	for _, b := range buffs {
		if b.Expired() {
			continue
		}
		for _, c := range b.Changes() {
			if c.Type() == string(statType) {
				return true
			}
		}
	}
	return false
}

// requiredClassification returns the projectile classification expected by the given
// ranged weapon type. The second return is false for non-ranged weapons.
func requiredClassification(w item.WeaponType) (item.Classification, bool) {
	switch w {
	case item.WeaponTypeBow, item.WeaponTypeCrossbow:
		return item.ClassificationConsumableArrow, true
	case item.WeaponTypeClaw:
		return item.ClassificationConsumableThrowingStar, true
	case item.WeaponTypeGun:
		return item.ClassificationBullet, true
	default:
		return 0, false
	}
}

// computeCount returns the total projectile quantity expected to be consumed for this
// attack. Base is the skill's BulletConsume when > 0, else 1. Claws under Shadow
// Partner double the total.
func computeCount(w item.WeaponType, se effect.Model, buffs []buff.Model) int {
	count := int(se.BulletConsume())
	if count <= 0 {
		count = 1
	}
	if w == item.WeaponTypeClaw && hasBuff(buffs, ts.TemporaryStatTypeShadowPartner) {
		count *= 2
	}
	return count
}

// resolvePlan picks slots in the consumable compartment to draw `count` projectiles
// from, matching `classification`. Preference order:
//  1. Client-suggested slot, if it has a matching item and enough quantity — one draw.
//  2. Lowest-index single slot that alone has enough quantity — one draw.
//  3. Multi-slot ascending-index draw until `count` is reached.
//  4. If combined available < count: consume everything matching; plan.Shortfall is true.
//
// The returned `available` is the sum of planned draws (== min(count, total matching)).
func resolvePlan(assets []asset.Model, classification item.Classification, clientSlot int16, count int) ([]ProjectileSlotDraw, int) {
	matching := make([]asset.Model, 0, len(assets))
	for _, a := range assets {
		if item.GetClassification(item.Id(a.TemplateId())) == classification && a.Quantity() > 0 {
			matching = append(matching, a)
		}
	}
	if len(matching) == 0 {
		return nil, 0
	}
	sort.Slice(matching, func(i, j int) bool { return matching[i].Slot() < matching[j].Slot() })

	// 1. Client-suggested slot hit.
	if clientSlot > 0 {
		for _, a := range matching {
			if a.Slot() == clientSlot && int(a.Quantity()) >= count {
				return []ProjectileSlotDraw{{Slot: a.Slot(), ItemId: a.TemplateId(), Quantity: int16(count)}}, count
			}
		}
	}

	// 2. Lowest-index single slot with enough.
	for _, a := range matching {
		if int(a.Quantity()) >= count {
			return []ProjectileSlotDraw{{Slot: a.Slot(), ItemId: a.TemplateId(), Quantity: int16(count)}}, count
		}
	}

	// 3+4. Multi-slot ascending draw, possibly short.
	remaining := count
	draws := make([]ProjectileSlotDraw, 0, len(matching))
	available := 0
	for _, a := range matching {
		if remaining <= 0 {
			break
		}
		draw := int(a.Quantity())
		if draw > remaining {
			draw = remaining
		}
		draws = append(draws, ProjectileSlotDraw{Slot: a.Slot(), ItemId: a.TemplateId(), Quantity: int16(draw)})
		remaining -= draw
		available += draw
	}
	return draws, available
}
