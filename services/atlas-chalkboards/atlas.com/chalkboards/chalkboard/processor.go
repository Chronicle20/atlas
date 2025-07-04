package chalkboard

import (
	chalkboard2 "atlas-chalkboards/kafka/message/chalkboard"
	"atlas-chalkboards/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(characterId uint32) (Model, error)
	Clear(field field.Model, characterId uint32) error
	Set(field field.Model, characterId uint32, message string) error
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

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	m, ok := getRegistry().Get(characterId)
	if !ok {
		return Model{}, errors.New("not found")
	}
	return Model{
		id:      characterId,
		message: m,
	}, nil
}

func (p *ProcessorImpl) Clear(field field.Model, characterId uint32) error {
	existed := getRegistry().Clear(characterId)
	if existed {
		p.l.Debugf("Clearing chalkboard for [%d].", characterId)
		return producer.ProviderImpl(p.l)(p.ctx)(chalkboard2.EnvEventTopicStatus)(clearStatusEventProvider(field, characterId))
	}
	return nil
}

func (p *ProcessorImpl) Set(field field.Model, characterId uint32, message string) error {
	// TODO ensure they are in a place that this can be set.
	// TODO ensure they are alive.
	p.l.Debugf("Setting chalkboard to [%s] for [%d].", message, characterId)
	getRegistry().Set(characterId, message)
	return producer.ProviderImpl(p.l)(p.ctx)(chalkboard2.EnvEventTopicStatus)(setStatusEventProvider(field, characterId, message))
}
