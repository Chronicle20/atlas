package rates

import (
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func byCharacterIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(worldId world.Id, channelId channel.Id, characterId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(worldId world.Id, channelId channel.Id, characterId uint32) model.Provider[Model] {
		return func(worldId world.Id, channelId channel.Id, characterId uint32) model.Provider[Model] {
			return requests.Provider[RestModel, Model](l, ctx)(requestForCharacter(worldId, channelId, characterId), Extract)
		}
	}
}

// GetForCharacter retrieves computed rates for a character
// Returns default rates (all 1.0) if the rate service is unavailable
func GetForCharacter(l logrus.FieldLogger) func(ctx context.Context) func(worldId world.Id, channelId channel.Id, characterId uint32) Model {
	return func(ctx context.Context) func(worldId world.Id, channelId channel.Id, characterId uint32) Model {
		return func(worldId world.Id, channelId channel.Id, characterId uint32) Model {
			m, err := byCharacterIdProvider(l)(ctx)(worldId, channelId, characterId)()
			if err != nil {
				l.WithError(err).Debugf("Unable to get rates for character [%d], using defaults.", characterId)
				return Default()
			}
			return m
		}
	}
}
