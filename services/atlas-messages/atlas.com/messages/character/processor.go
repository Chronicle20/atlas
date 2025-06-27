package character

import (
	"atlas-messages/kafka/message/character"
	"atlas-messages/kafka/producer"
	"atlas-messages/skill"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error)
	ByNameProvider(decorators ...model.Decorator[Model]) func(name string) model.Provider[[]Model]
	GetByName(decorators ...model.Decorator[Model]) func(name string) (Model, error)
	IdByNameProvider(name string) model.Provider[uint32]
	SkillModelDecorator(m Model) Model
	ChangeJob(worldId byte, channelId byte, characterId uint32, jobId uint16) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	sp  skill.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		sp:  skill.NewProcessor(l, ctx),
	}
	return p
}

func (p *ProcessorImpl) GetById(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		cp := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)
		return model.Map(model.Decorate(decorators))(cp)()
	}
}

func (p *ProcessorImpl) ByNameProvider(decorators ...model.Decorator[Model]) func(name string) model.Provider[[]Model] {
	return func(name string) model.Provider[[]Model] {
		ps := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByName(name), Extract, model.Filters[Model]())
		return model.SliceMap(model.Decorate(decorators))(ps)(model.ParallelMap())
	}
}

func (p *ProcessorImpl) GetByName(decorators ...model.Decorator[Model]) func(name string) (Model, error) {
	return func(name string) (Model, error) {
		return model.First(p.ByNameProvider(decorators...)(name), model.Filters[Model]())
	}
}

func (p *ProcessorImpl) IdByNameProvider(name string) model.Provider[uint32] {
	c, err := p.GetByName()(name)
	if err != nil {
		return model.ErrorProvider[uint32](err)
	}
	return model.FixedProvider(c.Id())
}

func (p *ProcessorImpl) SkillModelDecorator(m Model) Model {
	ms, err := p.sp.GetByCharacterId(m.Id())
	if err != nil {
		return m
	}
	return m.SetSkills(ms)
}

func (p *ProcessorImpl) ChangeJob(worldId byte, channelId byte, characterId uint32, jobId uint16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(character.EnvCommandTopic)(changeJobCommandProvider(characterId, worldId, channelId, jobId))
}
