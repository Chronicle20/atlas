package world

import (
	"atlas-world/channel"
	"atlas-world/configuration"
	"atlas-world/rate"
	"context"
	"errors"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

var errWorldNotFound = errors.New("world not found")

type Processor interface {
	ChannelDecorator(m Model) Model
	GetWorlds(decorators ...model.Decorator[Model]) ([]Model, error)
	AllWorldProvider(decorators ...model.Decorator[Model]) model.Provider[[]Model]
	GetWorld(decorators ...model.Decorator[Model]) func(worldId byte) (Model, error)
	ByWorldIdProvider(decorators ...model.Decorator[Model]) func(worldId byte) model.Provider[Model]
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	cp  channel.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		cp:  channel.NewProcessor(l, ctx),
	}
}

func (p *ProcessorImpl) ChannelDecorator(m Model) Model {
	cs, err := p.cp.GetByWorld(m.Id())
	if err != nil {
		return m
	}
	decorated, err := CloneModel(m).SetChannels(cs).Build()
	if err != nil {
		return m
	}
	return decorated
}

func (p *ProcessorImpl) AllWorldProvider(decorators ...model.Decorator[Model]) model.Provider[[]Model] {
	worldIds := mapDistinctWorldId(channel.GetChannelRegistry().ChannelServers(p.t))
	return model.SliceMap[byte, Model](func(b byte) (Model, error) {
		return p.ByWorldIdProvider(decorators...)(b)()
	})(model.FixedProvider[[]byte](worldIds))(model.ParallelMap())
}

func (p *ProcessorImpl) GetWorlds(decorators ...model.Decorator[Model]) ([]Model, error) {
	return p.AllWorldProvider(decorators...)()
}

func (p *ProcessorImpl) ByWorldIdProvider(decorators ...model.Decorator[Model]) func(worldId byte) model.Provider[Model] {
	return func(worldId byte) model.Provider[Model] {
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

		// Get current rates from registry (may have been updated at runtime)
		rates := rate.GetRegistry().GetWorldRates(p.t, worldId)

		m, err := NewModelBuilder().
			SetId(worldId).
			SetName(wc.Name).
			SetState(getFlag(wc.Flag)).
			SetMessage(wc.ServerMessage).
			SetEventMessage(wc.EventMessage).
			SetRecommendedMessage(wc.WhyAmIRecommended).
			SetCapacityStatus(0).
			SetExpRate(rates.ExpRate()).
			SetMesoRate(rates.MesoRate()).
			SetItemDropRate(rates.ItemDropRate()).
			SetQuestExpRate(rates.QuestExpRate()).
			Build()
		if err != nil {
			return model.ErrorProvider[Model](err)
		}
		return model.Map(model.Decorate(model.Decorators(decorators...)))(model.FixedProvider(m))
	}
}

func (p *ProcessorImpl) GetWorld(decorators ...model.Decorator[Model]) func(worldId byte) (Model, error) {
	return func(worldId byte) (Model, error) {
		return p.ByWorldIdProvider(decorators...)(worldId)()
	}
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

func getFlag(flag string) State {
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
