package monster

import (
	monster2 "atlas-channel/kafka/message/monster"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	monsterconst "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) GetById(uniqueId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(uniqueId), Extract)()
}

func (p *Processor) InMapModelProvider(f field.Model) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInMap(f), Extract, model.Filters[Model]())
}

func (p *Processor) ForEachInMap(f field.Model, o model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(f), o, model.ParallelExecute())
}

func (p *Processor) GetInMap(f field.Model) ([]Model, error) {
	return p.InMapModelProvider(f)()
}

// InMapRectModelProvider issues the atlas-monsters rect query and returns a
// provider over the resulting monsters. Used by AoE skill handlers (e.g.,
// Priest Doom) that need server-side bounding-box mob selection.
func (p *Processor) InMapRectModelProvider(f field.Model, x1, y1, x2, y2 int16, limit uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInMapRect(f, x1, y1, x2, y2, limit), Extract, model.Filters[Model]())
}

// GetInMapRect resolves the rect query into a slice of monsters.
func (p *Processor) GetInMapRect(f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]Model, error) {
	return p.InMapRectModelProvider(f, x1, y1, x2, y2, limit)()
}

func (p *Processor) Damage(f field.Model, monsterId uint32, characterId uint32, damages []uint32, attackType byte) error {
	p.l.Debugf("Applying damage to monster [%d]. Character [%d]. Lines [%d].", monsterId, characterId, len(damages))
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(DamageCommandProvider(f, monsterId, characterId, damages, attackType))
}

// EmitDamageReflected publishes a DAMAGE_REFLECTED status event so the
// existing monster-status consumer can apply the reflected damage to the
// attacker's HP. Called from the attack handler when a monster's reflect
// effect (PHYSICAL or MAGICAL) bounces a player attack. uniqueId is the
// spawned monster's unique id (same as mp.Damage's monsterId arg);
// templateId is the monster template id, required for the StatusEvent
// envelope. reflectType is the reflect kind (PHYSICAL/MAGICAL).
func (p *Processor) EmitDamageReflected(f field.Model, uniqueId uint32, templateId uint32, characterId uint32, reflectDamage uint32, reflectType string) error {
	p.l.Debugf("Emitting DAMAGE_REFLECTED for monster [%d] -> character [%d]. Reflect [%d] kind [%s].", uniqueId, characterId, reflectDamage, reflectType)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvEventTopicStatus)(DamageReflectedStatusEventProvider(f, uniqueId, templateId, characterId, reflectDamage, reflectType))
}

func (p *Processor) UseSkill(f field.Model, monsterId uint32, characterId uint32, skillId byte, skillLevel byte) error {
	p.l.Debugf("Monster [%d] using skill [%d] level [%d]. Controller [%d].", monsterId, skillId, skillLevel, characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(UseSkillCommandProvider(f, monsterId, characterId, skillId, skillLevel))
}

func (p *Processor) UseBasicAttack(f field.Model, monsterId uint32, attackPos uint8) error {
	p.l.Debugf("Monster [%d] using basic attack pos [%d].", monsterId, attackPos)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(UseBasicAttackCommandProvider(f, monsterId, attackPos))
}

func (p *Processor) ApplyStatus(f field.Model, monsterId uint32, characterId uint32, skillId uint32, skillLevel uint32, statuses map[string]int32, duration uint32) error {
	p.l.Debugf("Applying status to monster [%d]. Character [%d]. Skill [%d].", monsterId, characterId, skillId)
	if _, isDoom := statuses[monsterconst.StatusDoom]; isDoom {
		p.l.Debugf("Doom: caster=[%d] monster=[%d] skill=[%d] level=[%d] duration=[%d]ms.", characterId, monsterId, skillId, skillLevel, duration)
	}
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(ApplyStatusCommandProvider(f, monsterId, characterId, skillId, skillLevel, statuses, duration))
}

func (p *Processor) DamageFriendly(f field.Model, attackedUniqueId uint32, observerUniqueId, attackerUniqueId uint32) error {
	p.l.Debugf("Monster [%d] attacking friendly monster [%d].", attackerUniqueId, attackedUniqueId)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(DamageFriendlyCommandProvider(f, attackedUniqueId, observerUniqueId, attackerUniqueId))
}

func (p *Processor) CancelStatus(f field.Model, monsterId uint32, statusTypes []string, sourceCharacterId uint32, sourceSkillId uint32, sourceSkillClass string) error {
	p.l.Debugf("Cancelling status from monster [%d]. Types [%v]. Source character [%d] skill [%d] class [%s].", monsterId, statusTypes, sourceCharacterId, sourceSkillId, sourceSkillClass)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(CancelStatusCommandProvider(f, monsterId, statusTypes, sourceCharacterId, sourceSkillId, sourceSkillClass))
}

// DrainMp emits a DRAIN_MP command instructing atlas-monsters to deduct
// MP from a monster as the result of a player passive. The channel
// pre-screens cheap guards (MaxMp/Mp non-zero); atlas-monsters does the
// authoritative boss check and final clamp. The actual proc visual and
// caster MP refund are deferred to the MP_CHANGED return event.
func (p *Processor) DrainMp(f field.Model, monsterId uint32, characterId uint32, skillId uint32, amount uint32) error {
	p.l.Debugf("Draining MP from monster [%d] for character [%d] via skill [%d]. Amount [%d].", monsterId, characterId, skillId, amount)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(DrainMpCommandProvider(f, monsterId, characterId, skillId, amount))
}
