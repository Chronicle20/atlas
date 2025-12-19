package drop

import (
	drop2 "atlas-character/kafka/message/drop"
	"atlas-character/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	CreateForMesos(field field.Model, mesos uint32, dropType byte, x int16, y int16, ownerId uint32) error
	RequestPickUp(field field.Model, dropId uint32, characterId uint32) error
	CancelReservation(field field.Model, dropId uint32, characterId uint32) error
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

func (p *ProcessorImpl) CreateForMesos(field field.Model, mesos uint32, dropType byte, x int16, y int16, ownerId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(drop2.EnvCommandTopic)(dropMesoProvider(field, mesos, dropType, x, y, ownerId))
}

func (p *ProcessorImpl) RequestPickUp(field field.Model, dropId uint32, characterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(drop2.EnvCommandTopic)(requestPickUpCommandProvider(field, dropId, characterId))
}

func (p *ProcessorImpl) CancelReservation(field field.Model, dropId uint32, characterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(drop2.EnvCommandTopic)(cancelReservationCommandProvider(field, dropId, characterId))
}
