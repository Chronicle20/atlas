package character

import (
	"atlas-messages/inventory"
	character2 "atlas-messages/kafka/message/character"
	"atlas-messages/kafka/producer"
	"atlas-messages/skill"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	ip  *inventory.Processor
	sp  *skill.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		ip:  inventory.NewProcessor(l, ctx),
		sp:  skill.NewProcessor(l, ctx),
	}
	return p
}

func (p *Processor) GetById(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		cp := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)
		return model.Map(model.Decorate(decorators))(cp)()
	}
}

func (p *Processor) ByNameProvider(decorators ...model.Decorator[Model]) func(name string) model.Provider[[]Model] {
	return func(name string) model.Provider[[]Model] {
		ps := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByName(name), Extract, model.Filters[Model]())
		return model.SliceMap(model.Decorate(decorators))(ps)(model.ParallelMap())
	}
}

func (p *Processor) GetByName(decorators ...model.Decorator[Model]) func(name string) (Model, error) {
	return func(name string) (Model, error) {
		return model.First(p.ByNameProvider(decorators...)(name), model.Filters[Model]())
	}
}

func (p *Processor) IdByNameProvider(name string) model.Provider[uint32] {
	c, err := p.GetByName()(name)
	if err != nil {
		return model.ErrorProvider[uint32](err)
	}
	return model.FixedProvider(c.Id())
}

func (p *Processor) InventoryDecorator(m Model) Model {
	i, err := p.ip.GetByCharacterId(m.Id())
	if err != nil {
		return m
	}
	return m.SetInventory(i)
}

func (p *Processor) SkillModelDecorator(m Model) Model {
	ms, err := p.sp.GetByCharacterId(m.Id())
	if err != nil {
		return m
	}
	return m.SetSkills(ms)
}

func (p *Processor) AwardExperience(worldId byte, channelId byte, characterId uint32, amount uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(awardExperienceCommandProvider(characterId, worldId, channelId, amount))
}

func (p *Processor) AwardLevel(worldId byte, channelId byte, characterId uint32, amount byte) error {
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(awardLevelCommandProvider(characterId, worldId, channelId, amount))
}

func (p *Processor) ChangeJob(worldId byte, channelId byte, characterId uint32, jobId uint16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(changeJobCommandProvider(characterId, worldId, channelId, jobId))
}
