package rate

import (
	"atlas-world/kafka/message"
	rateMessage "atlas-world/kafka/message/rate"
	"atlas-world/kafka/producer"
	rateProducer "atlas-world/kafka/producer/rate"
	"context"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetWorldRates(worldId byte) Model
	UpdateWorldRate(worldId byte, rateType Type, multiplier float64) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
}

func (p *ProcessorImpl) GetWorldRates(worldId byte) Model {
	return GetRegistry().GetWorldRates(p.t, worldId)
}

func (p *ProcessorImpl) UpdateWorldRate(worldId byte, rateType Type, multiplier float64) error {
	p.l.Debugf("Updating world [%d] rate [%s] to [%.2f] for tenant [%s].", worldId, rateType, multiplier, p.t.String())

	GetRegistry().SetWorldRate(p.t, worldId, rateType, multiplier)

	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return buf.Put(rateMessage.EnvEventTopicWorldRate, rateProducer.WorldRateChangedEventProvider(p.t, worldId, rateMessage.RateType(rateType), multiplier))
	})
}
