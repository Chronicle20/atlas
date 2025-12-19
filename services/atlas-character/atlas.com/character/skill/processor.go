package skill

import (
	skill2 "atlas-character/kafka/message/skill"
	"atlas-character/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"time"
)

type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacterId(characterId uint32) ([]Model, error)
	RequestCreate(characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error
	RequestUpdate(characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) RequestCreate(characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
	return producer.ProviderImpl(p.l)(p.ctx)(skill2.EnvCommandTopic)(createCommandProvider(characterId, skillId, level, masterLevel, expiration))
}

func (p *ProcessorImpl) RequestUpdate(characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
	return producer.ProviderImpl(p.l)(p.ctx)(skill2.EnvCommandTopic)(updateCommandProvider(characterId, skillId, level, masterLevel, expiration))
}
