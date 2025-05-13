package fame

import (
	"atlas-fame/character"
	"atlas-fame/database"
	"atlas-fame/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"time"
)

func byCharacterIdLastMonthProvider(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
			return func(characterId uint32) model.Provider[[]Model] {
				return model.SliceMap(Make)(byCharacterIdLastMonthEntityProvider(t.Id(), characterId)(db))()
			}
		}
	}
}

func GetByCharacterIdLastMonth(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) ([]Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) ([]Model, error) {
		return func(db *gorm.DB) func(characterId uint32) ([]Model, error) {
			return func(characterId uint32) ([]Model, error) {
				return byCharacterIdLastMonthProvider(l)(ctx)(db)(characterId)()
			}
		}
	}
}

func RequestChange(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, channelId byte, characterId uint32, mapId uint32, targetId uint32, amount int8) error {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, channelId byte, characterId uint32, mapId uint32, targetId uint32, amount int8) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(worldId byte, channelId byte, characterId uint32, mapId uint32, targetId uint32, amount int8) error {
			return func(worldId byte, channelId byte, characterId uint32, mapId uint32, targetId uint32, amount int8) error {
				return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
					c, err := character.GetById(l)(ctx)(characterId)
					if err != nil {
						_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicFameStatus)(errorEventStatusProvider(worldId, channelId, characterId, StatusEventErrorTypeUnexpected))
						return nil
					}

					_, err = character.GetById(l)(ctx)(targetId)
					if err != nil {
						_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicFameStatus)(errorEventStatusProvider(worldId, channelId, characterId, StatusEventErrorInvalidName))
						return nil
					}

					if c.Level() < 15 {
						_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicFameStatus)(errorEventStatusProvider(worldId, channelId, characterId, StatusEventErrorTypeNotMinimumLevel))
						return nil
					}

					fls, err := GetByCharacterIdLastMonth(l)(ctx)(tx)(characterId)
					if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
						return err
					}

					famedToday := false
					famedTargetLastMonth := false
					now := time.Now()
					for _, fl := range fls {
						if fl.TargetId() == targetId {
							famedTargetLastMonth = true
						}
						if fl.CreatedAt().Year() == now.Year() && fl.CreatedAt().YearDay() == now.YearDay() {
							famedToday = true
						}
					}
					if famedToday {
						_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicFameStatus)(errorEventStatusProvider(worldId, channelId, characterId, StatusEventErrorTypeNotToday))
						return nil
					}
					if famedTargetLastMonth {

						_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicFameStatus)(errorEventStatusProvider(worldId, channelId, characterId, StatusEventErrorTypeNotThisMonth))
						return nil
					}
					_, err = create(tx, t, characterId, targetId, amount)
					if err != nil {
						_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicFameStatus)(errorEventStatusProvider(worldId, channelId, characterId, StatusEventErrorTypeUnexpected))
						return err
					}

					return character.RequestChangeFame(l)(ctx)(targetId, worldId, characterId, amount)
				})
			}
		}
	}
}
