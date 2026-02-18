package map_command

import (
	mapKafka "atlas-saga-orchestrator/kafka/message/map"
	"atlas-saga-orchestrator/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	FieldEffectWeather(transactionId uuid.UUID, f field.Model, itemId uint32, message string, durationMs uint32) error
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

func (p *ProcessorImpl) FieldEffectWeather(transactionId uuid.UUID, f field.Model, itemId uint32, message string, durationMs uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(mapKafka.EnvCommandTopicMap)(WeatherStartCommandProvider(transactionId, f, itemId, message, durationMs))
}
