package rates

import (
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func byCharacterIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(ch channel.Model, characterId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(ch channel.Model, characterId uint32) model.Provider[Model] {
		return func(ch channel.Model, characterId uint32) model.Provider[Model] {
			return func() (Model, error) {
				resp, err := requestRates(ch, characterId)(l, ctx)
				if err != nil {
					return Model{}, err
				}
				return Extract(resp.Data), nil
			}
		}
	}
}

func GetForCharacter(l logrus.FieldLogger) func(ctx context.Context) func(ch channel.Model, characterId uint32) Model {
	return func(ctx context.Context) func(ch channel.Model, characterId uint32) Model {
		return func(ch channel.Model, characterId uint32) Model {
			m, err := byCharacterIdProvider(l)(ctx)(ch, characterId)()
			if err != nil {
				l.WithError(err).Warnf("Unable to get rates for character [%d], using defaults", characterId)
				return Default()
			}
			return m
		}
	}
}
