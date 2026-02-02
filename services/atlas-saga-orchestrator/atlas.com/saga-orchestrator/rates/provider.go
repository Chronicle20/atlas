package rates

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func byCharacterIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) model.Provider[Model] {
		return func(worldId byte, channelId byte, characterId uint32) model.Provider[Model] {
			return func() (Model, error) {
				resp, err := requestRates(worldId, channelId, characterId)(l, ctx)
				if err != nil {
					return Model{}, err
				}
				return Extract(resp.Data), nil
			}
		}
	}
}

func GetForCharacter(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) Model {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) Model {
		return func(worldId byte, channelId byte, characterId uint32) Model {
			m, err := byCharacterIdProvider(l)(ctx)(worldId, channelId, characterId)()
			if err != nil {
				l.WithError(err).Warnf("Unable to get rates for character [%d], using defaults", characterId)
				return Default()
			}
			return m
		}
	}
}
