package portal

import (
	portalData "atlas-channel/data/portal"
	"atlas-channel/kafka/message/portal"
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Enter(m _map.Model, portalName string, characterId uint32) error
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

func (p *ProcessorImpl) Enter(m _map.Model, portalName string, characterId uint32) error {
	pm, err := p.pd.GetInMapByName(m.MapId(), portalName)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate portal [%s] in map [%d].", portalName, m.MapId())
		return err
	}
	err = producer.ProviderImpl(p.l)(p.ctx)(portal.EnvPortalCommandTopic)(EnterCommandProvider(m, pm.Id(), characterId))
	return err
}
