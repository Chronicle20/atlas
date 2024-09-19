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

func AllWorldProvider(ctx context.Context) model.Provider[[]Model] {
	t := tenant.MustFromContext(ctx)
	worldIds := mapDistinctWorldId(channel.GetChannelRegistry().ChannelServers(t.Id().String()))
	return model.SliceMap[byte, Model](worldTransformer(ctx))(model.FixedProvider[[]byte](worldIds))(model.ParallelMap())
}

func GetWorlds(_ logrus.FieldLogger) func(ctx context.Context) ([]Model, error) {
	return func(ctx context.Context) ([]Model, error) {
		return AllWorldProvider(ctx)()
	}
}

func worldTransformer(ctx context.Context) func(b byte) (Model, error) {
	return func(b byte) (Model, error) {
		return ByWorldIdProvider(ctx)(b)()
	}
}

func ByWorldIdProvider(ctx context.Context) func(worldId byte) model.Provider[Model] {
	return func(worldId byte) model.Provider[Model] {
		t := tenant.MustFromContext(ctx)
		worldIds := mapDistinctWorldId(channel.GetChannelRegistry().ChannelServers(t.Id().String()))
		var exists = false
		for _, wid := range worldIds {
			if wid == worldId {
				exists = true
			}
		}
		if !exists {
			return model.ErrorProvider[Model](errWorldNotFound)
		}

		c, err := configuration.GetConfiguration()
		if err != nil {
			return model.ErrorProvider[Model](err)
		}

		wc, err := c.FindWorld(t.Id().String(), worldId)
		if err != nil {
			return model.ErrorProvider[Model](err)
		}
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
}

func GetWorld(_ logrus.FieldLogger) func(ctx context.Context) func(worldId byte) (Model, error) {
	return func(ctx context.Context) func(worldId byte) (Model, error) {
		return func(worldId byte) (Model, error) {
			return ByWorldIdProvider(ctx)(worldId)()
		}
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
