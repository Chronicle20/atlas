package map_command

import (
	mapKafka "atlas-saga-orchestrator/kafka/message/map"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type Processor interface {
	FieldEffectWeather(transactionId uuid.UUID, f field.Model, itemId uint32, message string, durationMs uint32) error
	PlayJukebox(transactionId uuid.UUID, f field.Model, itemId uint32, playerName string, durationMs uint32) error
	SetEnvironmentState(transactionId uuid.UUID, f field.Model, kind field.ObjectKind, name string, state uint32) error
	ResetEnvironment(transactionId uuid.UUID, f field.Model) error
	SetBackEffect(transactionId uuid.UUID, f field.Model, effect uint8, fieldId uint32, pageId uint8, duration uint32) error
	ClearBackEffect(transactionId uuid.UUID, f field.Model) error
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) FieldEffectWeather(transactionId uuid.UUID, f field.Model, itemId uint32, message string, durationMs uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(mapKafka.EnvCommandTopicMap)(WeatherStartCommandProvider(transactionId, f, itemId, message, durationMs))
}

func (p *ProcessorImpl) PlayJukebox(transactionId uuid.UUID, f field.Model, itemId uint32, playerName string, durationMs uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(mapKafka.EnvCommandTopicMap)(PlayJukeboxCommandProvider(transactionId, f, itemId, playerName, durationMs))
}

func (p *ProcessorImpl) SetEnvironmentState(transactionId uuid.UUID, f field.Model, kind field.ObjectKind, name string, state uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(mapKafka.EnvCommandTopicMap)(SetEnvironmentStateCommandProvider(transactionId, f, kind, name, state))
}

func (p *ProcessorImpl) ResetEnvironment(transactionId uuid.UUID, f field.Model) error {
	return producer.ProviderImpl(p.l)(p.ctx)(mapKafka.EnvCommandTopicMap)(ResetEnvironmentCommandProvider(transactionId, f))
}

func (p *ProcessorImpl) SetBackEffect(transactionId uuid.UUID, f field.Model, effect uint8, fieldId uint32, pageId uint8, duration uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(mapKafka.EnvCommandTopicMap)(SetBackEffectCommandProvider(transactionId, f, effect, fieldId, pageId, duration))
}

func (p *ProcessorImpl) ClearBackEffect(transactionId uuid.UUID, f field.Model) error {
	return producer.ProviderImpl(p.l)(p.ctx)(mapKafka.EnvCommandTopicMap)(ClearBackEffectCommandProvider(transactionId, f))
}
