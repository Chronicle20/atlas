// Package saga submits fully-built libs/atlas-saga Saga values to
// atlas-saga-orchestrator's command topic. It is the composition-root-facing
// counterpart of game.SagaSubmitter: game/processor.go builds the payout
// Saga value (importing libs/atlas-saga directly, which has no dependency on
// atlas-rps) and hands it to an injected SagaSubmitter closure; production
// wiring (main.go, kafka/consumer/rps) backs that closure with
// saga.NewProcessor(l, ctx).Create(s) from this package. Mirrors
// atlas-npc-conversations/atlas.com/npc/saga/processor.go.
package saga

import (
	"atlas-rps/kafka/message/saga"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// Processor submits a Saga to atlas-saga-orchestrator's command topic.
type Processor interface {
	Create(s sharedsaga.Saga) error
}

// ProcessorImpl implements the Processor interface.
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new processor implementation.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// Create submits s to atlas-saga-orchestrator's command topic.
func (p *ProcessorImpl) Create(s sharedsaga.Saga) error {
	return producer.ProviderImpl(p.l)(p.ctx)(saga.EnvCommandTopic)(createCommandProvider(s))
}
