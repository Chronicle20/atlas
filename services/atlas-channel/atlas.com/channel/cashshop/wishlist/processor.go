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

func SetForCharacter(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, items []uint32) error {
	return func(ctx context.Context) func(characterId uint32, items []uint32) error {
		return func(characterId uint32, serialNumbers []uint32) error {
			l.Debugf("Setting wishlist for character [%d].", characterId)
			err := clearForCharacterId(characterId)(l, ctx)
			if err != nil {
				l.WithError(err).Errorf("Unable to clear wishlist for character [%d].", characterId)
				return err
			}
			for _, serialNumber := range serialNumbers {
				if serialNumber == 0 {
					continue
				}
				_, err = addForCharacterId(characterId, serialNumber)(l, ctx)
				if err != nil {
					l.WithError(err).Errorf("Unable to add serialNumber [%d] to wishlist for character [%d].", serialNumber, characterId)
				}
			}
			return nil
		}
	}
}
