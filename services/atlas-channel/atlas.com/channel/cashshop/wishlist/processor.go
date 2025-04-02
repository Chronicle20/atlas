package wishlist

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func byCharacterIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
		return func(characterId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestByCharacterId(characterId), Extract, model.Filters[Model]())
		}
	}
}

func GetByCharacterId(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) ([]Model, error) {
	return func(ctx context.Context) func(characterId uint32) ([]Model, error) {
		return func(characterId uint32) ([]Model, error) {
			return byCharacterIdProvider(l)(ctx)(characterId)()
		}
	}
}

func SetForCharacter(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, items []uint32) ([]Model, error) {
	return func(ctx context.Context) func(characterId uint32, items []uint32) ([]Model, error) {
		return func(characterId uint32, serialNumbers []uint32) ([]Model, error) {
			l.Debugf("Setting wishlist for character [%d].", characterId)
			results := make([]Model, 0)
			err := clearForCharacterId(characterId)(l, ctx)
			if err != nil {
				l.WithError(err).Errorf("Unable to clear wishlist for character [%d].", characterId)
				return results, err
			}
			for _, serialNumber := range serialNumbers {
				if serialNumber == 0 {
					continue
				}
				var rm RestModel
				rm, err = addForCharacterId(characterId, serialNumber)(l, ctx)
				if err != nil {
					l.WithError(err).Errorf("Unable to add serialNumber [%d] to wishlist for character [%d].", serialNumber, characterId)
					continue
				}
				var m Model
				m, err = Extract(rm)
				if err != nil {
					l.WithError(err).Errorf("Unable to extract wishlist item for character [%d].", characterId)
				}
				results = append(results, m)
			}
			return results, nil
		}
	}
}
