package skill

import (
	"atlas-skills/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"time"
)

func byCharacterIdProvider(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
		return func(characterId uint32) model.Provider[[]Model] {
			return model.SliceMap(Make)(getByCharacterId(t.Id(), characterId)(db))()
		}
	}
}

func GetByCharacterId(ctx context.Context) func(db *gorm.DB) func(characterId uint32) ([]Model, error) {
	return func(db *gorm.DB) func(characterId uint32) ([]Model, error) {
		return func(characterId uint32) ([]Model, error) {
			return byCharacterIdProvider(ctx)(db)(characterId)()
		}
	}
}

func byIdProvider(ctx context.Context) func(db *gorm.DB) func(id uint32, threadId uint32) model.Provider[Model] {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(id uint32, threadId uint32) model.Provider[Model] {
		return func(id uint32, threadId uint32) model.Provider[Model] {
			return model.Map(Make)(getById(t.Id(), id, threadId)(db))
		}
	}
}

func GetById(ctx context.Context) func(db *gorm.DB) func(characterId uint32, id uint32) (Model, error) {
	return func(db *gorm.DB) func(characterId uint32, id uint32) (Model, error) {
		return func(characterId uint32, id uint32) (Model, error) {
			return byIdProvider(ctx)(db)(characterId, id)()
		}
	}
}

func RequestCreate(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
	return func(ctx context.Context) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
		return func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(createCommandProvider(characterId, id, level, masterLevel, expiration))
		}
	}
}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
			return func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
				l.Debugf("Attempting to create skill [%d] for character [%d].", id, characterId)
				var s Model
				txErr := db.Transaction(func(tx *gorm.DB) error {
					var err error
					s, err = GetById(ctx)(tx)(characterId, id)
					if s.Id() != 0 {
						return errors.New("already exists")
					}
					s, err = create(tx, t.Id(), characterId, id, level, masterLevel, expiration)
					if err != nil {
						return err
					}
					return nil
				})
				if txErr != nil {
					return Model{}, txErr
				}
				l.Debugf("Created skill [%d] for character [%d].", id, characterId)
				_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventCreatedProvider(characterId, s.Id(), s.Level(), s.MasterLevel(), s.Expiration()))
				return s, nil
			}
		}
	}
}

func RequestUpdate(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
	return func(ctx context.Context) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
		return func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(updateCommandProvider(characterId, id, level, masterLevel, expiration))
		}
	}
}

func Update(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
			return func(characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
				l.Debugf("Attempting to update skill [%d] for character [%d].", id, characterId)
				var s Model
				txErr := db.Transaction(func(tx *gorm.DB) error {
					var err error
					s, err = GetById(ctx)(tx)(characterId, id)
					if err != nil {
						return errors.New("does not exist")
					}
					err = dynamicUpdate(tx)(SetLevel(level), SetMasterLevel(masterLevel), SetExpiration(expiration))(t.Id(), characterId)(s)
					if err != nil {
						return err
					}
					s, err = GetById(ctx)(tx)(characterId, id)
					if err != nil {
						return errors.New("does not exist")
					}
					return nil
				})
				if txErr != nil {
					return Model{}, txErr
				}
				l.Debugf("Update skill [%d] for character [%d].", id, characterId)
				_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventUpdatedProvider(characterId, s.Id(), s.Level(), s.MasterLevel(), s.Expiration()))
				return s, nil
			}
		}
	}
}
