package portal

import (
	portalData "atlas-channel/data/portal"
	"atlas-channel/kafka/message/portal"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Enter(f field.Model, portalName string, characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	pd  portalData.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *ProcessorImpl {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		pd:  portalData.NewProcessor(l, ctx),
	}
	return p
}

func (p *ProcessorImpl) Enter(f field.Model, portalName string, characterId uint32) error {
	pm, err := p.pd.GetInMapByName(f.MapId(), portalName)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate portal [%s] in map [%d].", portalName, f.MapId())
		return err
	}
	err = producer.ProviderImpl(p.l)(p.ctx)(portal.EnvPortalCommandTopic)(EnterCommandProvider(f, pm.Id(), characterId))
	return err
}
