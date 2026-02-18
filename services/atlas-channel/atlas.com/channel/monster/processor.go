package monster

import (
	monster2 "atlas-channel/kafka/message/monster"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
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

func (p *Processor) Damage(f field.Model, monsterId uint32, characterId uint32, damage uint32, attackType byte) error {
	p.l.Debugf("Applying damage to monster [%d]. Character [%d]. Damage [%d].", monsterId, characterId, damage)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(DamageCommandProvider(f, monsterId, characterId, damage, attackType))
}

func (p *Processor) UseSkill(f field.Model, monsterId uint32, characterId uint32, skillId uint16, skillLevel uint16) error {
	p.l.Debugf("Monster [%d] using skill [%d] level [%d]. Controller [%d].", monsterId, skillId, skillLevel, characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(UseSkillCommandProvider(f, monsterId, characterId, skillId, skillLevel))
}

func (p *Processor) ApplyStatus(f field.Model, monsterId uint32, characterId uint32, skillId uint32, skillLevel uint32, statuses map[string]int32, duration uint32) error {
	p.l.Debugf("Applying status to monster [%d]. Character [%d]. Skill [%d].", monsterId, characterId, skillId)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(ApplyStatusCommandProvider(f, monsterId, characterId, skillId, skillLevel, statuses, duration))
}

func (p *Processor) DamageFriendly(f field.Model, attackedUniqueId uint32, observerUniqueId, attackerUniqueId uint32) error {
	p.l.Debugf("Monster [%d] attacking friendly monster [%d].", attackerUniqueId, attackedUniqueId)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(DamageFriendlyCommandProvider(f, attackedUniqueId, observerUniqueId, attackerUniqueId))
}

func (p *Processor) CancelStatus(f field.Model, monsterId uint32, statusTypes []string) error {
	p.l.Debugf("Cancelling status from monster [%d]. Types [%v].", monsterId, statusTypes)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(CancelStatusCommandProvider(f, monsterId, statusTypes))
}
