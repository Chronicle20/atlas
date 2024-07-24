package world

import (
	"atlas-world/channel"
	"atlas-world/configuration"
	"atlas-world/tenant"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

var errWorldNotFound = errors.New("world not found")

func allWorldProvider(l logrus.FieldLogger, tenant tenant.Model) model.Provider[[]Model] {
	worldIds := mapDistinctWorldId(channel.GetChannelRegistry().ChannelServers(tenant.Id.String()))
	return model.SliceMap[byte, Model](model.FixedProvider[[]byte](worldIds), worldTransformer(l, tenant))
}

func GetWorlds(l logrus.FieldLogger, tenant tenant.Model) ([]Model, error) {
	return allWorldProvider(l, tenant)()
}

func worldTransformer(l logrus.FieldLogger, tenant tenant.Model) func(b byte) (Model, error) {
	return func(b byte) (Model, error) {
		return byWorldIdProvider(l, tenant)(b)()
	}
}

func byWorldIdProvider(l logrus.FieldLogger, tenant tenant.Model) func(worldId byte) model.Provider[Model] {
	return func(worldId byte) model.Provider[Model] {
		worldIds := mapDistinctWorldId(channel.GetChannelRegistry().ChannelServers(tenant.Id.String()))
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

		wc, err := c.FindWorld(tenant.Id.String(), worldId)
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

func GetWorld(l logrus.FieldLogger, tenant tenant.Model) func(worldId byte) (Model, error) {
	return func(worldId byte) (Model, error) {
		return byWorldIdProvider(l, tenant)(worldId)()
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
