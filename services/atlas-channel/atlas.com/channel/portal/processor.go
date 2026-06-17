package portal

import (
	portalData "atlas-channel/data/portal"
	"atlas-channel/kafka/message/portal"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Enter(f field.Model, portalName string, characterId uint32) error
	Warp(f field.Model, characterId uint32, targetMapId _map.Id) error
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
	return producer.ProviderImpl(p.l)(p.ctx)(portal.EnvPortalCommandTopic)(EnterCommandProvider(f, pm.Id(), characterId))
}

func (p *ProcessorImpl) Warp(f field.Model, characterId uint32, targetMapId _map.Id) error {
	return producer.ProviderImpl(p.l)(p.ctx)(portal.EnvPortalCommandTopic)(WarpCommandProvider(f, characterId, targetMapId))
}

// WarpToPortal warps the character to a specific portal in the target map. A
// targetPortalId of 0 falls back to the random-spawn Warp.
func (p *ProcessorImpl) WarpToPortal(f field.Model, characterId uint32, targetMapId _map.Id, targetPortalId uint32) error {
	if targetPortalId == 0 {
		return p.Warp(f, characterId, targetMapId)
	}
	return producer.ProviderImpl(p.l)(p.ctx)(portal.EnvPortalCommandTopic)(WarpToPortalCommandProvider(f, characterId, targetMapId, targetPortalId))
}
