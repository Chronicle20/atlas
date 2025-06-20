package world

import (
	"atlas-world/channel"
	"atlas-world/configuration"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

var errWorldNotFound = errors.New("world not found")

type Processor interface {
	GetWorlds() ([]Model, error)
	AllWorldProvider() model.Provider[[]Model]
	GetWorld(worldId byte) (Model, error)
	ByWorldIdProvider(worldId byte) model.Provider[Model]
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

func (p *ProcessorImpl) AllWorldProvider() model.Provider[[]Model] {
	worldIds := mapDistinctWorldId(channel.GetChannelRegistry().ChannelServers(p.t))
	return model.SliceMap[byte, Model](p.worldTransformer)(model.FixedProvider[[]byte](worldIds))(model.ParallelMap())
}

func (p *ProcessorImpl) GetWorlds() ([]Model, error) {
	return p.AllWorldProvider()()
}

func (p *ProcessorImpl) worldTransformer(b byte) (Model, error) {
	return p.ByWorldIdProvider(b)()
}

func (p *ProcessorImpl) ByWorldIdProvider(worldId byte) model.Provider[Model] {
	worldIds := mapDistinctWorldId(channel.GetChannelRegistry().ChannelServers(p.t))
	var exists = false
	for _, wid := range worldIds {
		if wid == worldId {
			exists = true
		}
	}
	if !exists {
		return model.ErrorProvider[Model](errWorldNotFound)
	}

	c, err := configuration.GetTenantConfig(p.t.Id())
	if err != nil {
		return model.ErrorProvider[Model](err)
	}

	if len(c.Worlds) <= 0 || int(worldId) >= len(c.Worlds) {
		return model.ErrorProvider[Model](errors.New("world not found"))
	}
	wc := c.Worlds[worldId]
	m := Model{
		id:                 worldId,
		name:               wc.Name,
		flag:               wc.Flag,
		message:            wc.ServerMessage,
		eventMessage:       wc.EventMessage,
		recommendedMessage: wc.WhyAmIRecommended,
		capacityStatus:     0,
	}
	return model.FixedProvider[Model](m)
}

func (p *ProcessorImpl) GetWorld(worldId byte) (Model, error) {
	return p.ByWorldIdProvider(worldId)()
}

func mapDistinctWorldId(channelServers []channel.Model) []byte {
	m := make(map[byte]struct{})
	for _, element := range channelServers {
		m[element.WorldId()] = struct{}{}
	}

	keys := make([]byte, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getFlag(flag string) int {
	switch flag {
	case "NOTHING":
		return 0
	case "EVENT":
		return 1
	case "NEW":
		return 2
	case "HOT":
		return 3
	default:
		return 0
	}
}
