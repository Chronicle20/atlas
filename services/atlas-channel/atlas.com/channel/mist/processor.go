package mist

import (
	mistmsg "atlas-channel/kafka/message/mist"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

// Processor emits mist lifecycle commands to atlas-maps.
type Processor interface {
	Create(body mistmsg.CreateCommandBody) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// Create asks atlas-maps to spawn a mist. atlas-maps is authoritative for the
// mist's identity and lifecycle; this is fire-and-forget.
func (p *ProcessorImpl) Create(body mistmsg.CreateCommandBody) error {
	return producer.ProviderImpl(p.l)(p.ctx)(mistmsg.EnvCommandTopic)(CreateCommandProvider(body))
}
