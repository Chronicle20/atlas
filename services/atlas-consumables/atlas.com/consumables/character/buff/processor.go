package buff

import (
	"atlas-consumables/character/buff/stat"
	buff2 "atlas-consumables/kafka/message/character/buff"
	"context"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type Processor interface {
	Apply(f field.Model, fromId uint32, sourceId int32, level byte, duration int32, statups []stat.Model) model.Operator[uint32]
	Cancel(f field.Model, characterId uint32, sourceId int32) error
	CancelByTypes(f field.Model, characterId uint32, types []string) error
	GetByCharacterId(characterId uint32) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Apply(f field.Model, fromId uint32, sourceId int32, level byte, duration int32, statups []stat.Model) model.Operator[uint32] {
	return func(characterId uint32) error {
		return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(applyCommandProvider(f, characterId, fromId, sourceId, level, duration, statups))
	}
}

func (p *ProcessorImpl) Cancel(f field.Model, characterId uint32, sourceId int32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(cancelCommandProvider(f, characterId, sourceId))
}

func (p *ProcessorImpl) CancelByTypes(f field.Model, characterId uint32, types []string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(cancelByTypesCommandProvider(f, characterId, types))
}

// GetByCharacterId fetches every buff for a character. The upstream
// atlas-buffs list is paginated, so this drains every page rather than
// fetching just the first.
//
// A 404 is normalized to the empty set rather than propagated. atlas-buffs
// materializes a character's buff registry entry lazily, so GET
// /characters/{id}/buffs replies 404 until something applies a first buff --
// that is "this character has no buffs", not a failure.
func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	url, err := characterBuffsUrl(p.ctx, characterId)
	if err != nil {
		return nil, err
	}
	ms, err := requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model]())()
	if errors.Is(err, requests.ErrNotFound) {
		return []Model{}, nil
	}
	return ms, err
}
