package key

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var defaultKey = []int32{18, 65, 2, 23, 3, 4, 5, 6, 16, 17, 19, 25, 26, 27, 31, 34, 35, 37, 38, 40, 43, 44, 45, 46, 50, 56, 59, 60, 61, 62, 63, 64, 57, 48, 29, 7, 24, 33, 41, 39}
var defaultType = []int8{4, 6, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5, 5, 4, 4, 5, 6, 6, 6, 6, 6, 6, 5, 4, 5, 4, 4, 4, 4, 4}
var defaultAction = []int32{0, 106, 10, 1, 12, 13, 18, 24, 8, 5, 4, 19, 14, 15, 2, 17, 11, 3, 20, 16, 9, 50, 51, 6, 7, 53, 100, 101, 102, 103, 104, 105, 54, 22, 52, 21, 25, 26, 23, 27}

var entityModelMapper = model.Map(makeKey)
var entitySliceMapper = model.SliceMap(makeKey)

func makeKey(e entity) (Model, error) {
	return Model{
		characterId: e.CharacterId,
		key:         e.Key,
		theType:     e.Type,
		action:      e.Action,
	}, nil
}

func byCharacterIdProvider(db *gorm.DB) func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
		return func(characterId uint32) model.Provider[[]Model] {
			t := tenant.MustFromContext(ctx)
			return entitySliceMapper(byCharacterIdEntityProvider(t.Id(), characterId)(db))()
		}
	}
}

func GetByCharacterId(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) ([]Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) ([]Model, error) {
		return func(db *gorm.DB) func(characterId uint32) ([]Model, error) {
			return func(characterId uint32) ([]Model, error) {
				l.Debugf("Retrieving key map for character [%d].", characterId)
				return byCharacterIdProvider(db)(ctx)(characterId)()
			}
		}
	}
}

func Reset(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) error {
		return func(db *gorm.DB) func(characterId uint32) error {
			return func(characterId uint32) error {
				t := tenant.MustFromContext(ctx)
				return db.Transaction(func(tx *gorm.DB) error {
					err := deleteByCharacter(tx, t, characterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to delete for character %d.", characterId)
						return err
					}
					for i := 0; i < len(defaultKey); i++ {
						_, err := create(tx, t, characterId, defaultKey[i], defaultType[i], defaultAction[i])
						if err != nil {
							l.WithError(err).Errorf("Unable to create key binding for character %d. key = %d type = %d action = %d.", characterId, defaultKey[i], defaultType[i], defaultAction[i])
							return err
						}
					}
					return nil
				})
			}
		}
	}
}

func CreateDefault(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) error {
		return func(db *gorm.DB) func(characterId uint32) error {
			return func(characterId uint32) error {
				t := tenant.MustFromContext(ctx)
				return db.Transaction(func(tx *gorm.DB) error {
					for i := 0; i < len(defaultKey); i++ {
						_, err := create(tx, t, characterId, defaultKey[i], defaultType[i], defaultAction[i])
						if err != nil {
							l.WithError(err).Errorf("Unable to create key binding for character %d. key = %d type = %d action = %d.", characterId, defaultKey[i], defaultType[i], defaultAction[i])
							return err
						}
					}
					return nil
				})
			}
		}
	}
}

func ChangeKey(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, key int32, theType int8, action int32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, key int32, theType int8, action int32) error {
		return func(db *gorm.DB) func(characterId uint32, key int32, theType int8, action int32) error {
			return func(characterId uint32, key int32, theType int8, action int32) error {
				t := tenant.MustFromContext(ctx)
				return db.Transaction(func(tx *gorm.DB) error {
					_, err := byCharacterKeyEntityProvider(t.Id(), characterId, key)(tx)()
					if err != nil {
						_, err = create(tx, t, characterId, key, theType, action)
						if err != nil {
							l.WithError(err).Errorf("Unable to create key binding for character %d. key = %d type = %d action = %d.", characterId, key, theType, action)
							return err
						}
					} else {
						err = update(tx, t, characterId, key, theType, action)
						if err != nil {
							l.WithError(err).Errorf("Unable to update key binding for character %d. key = %d type = %d action = %d.", characterId, key, theType, action)
							return err
						}
					}
					return nil
				})
			}
		}
	}
}
