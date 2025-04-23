package portal

import (
	portal2 "atlas-channel/kafka/message/portal"
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) InMapByNameModelProvider(mapId _map.Id, name string) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInMapByName(mapId, name), Extract, model.Filters[Model]())
}

func (p *Processor) GetInMapByName(mapId _map.Id, name string) (Model, error) {
	return model.First(p.InMapByNameModelProvider(mapId, name), model.Filters[Model]())
}

func (p *Processor) Enter(m _map.Model, portalName string, characterId uint32) error {
	pm, err := p.GetInMapByName(m.MapId(), portalName)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate portal [%s] in map [%d].", portalName, m.MapId())
		return err
	}
	err = producer.ProviderImpl(p.l)(p.ctx)(portal2.EnvPortalCommandTopic)(EnterCommandProvider(m, pm.id, characterId))
	return err
}
