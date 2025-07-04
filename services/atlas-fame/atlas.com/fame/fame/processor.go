package fame

import (
	"atlas-fame/character"
	"atlas-fame/database"
	"atlas-fame/kafka/message"
	messageFame "atlas-fame/kafka/message/fame"
	"atlas-fame/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"time"
)

type Processor interface {
	// GetByCharacterIdLastMonth gets all fame logs for a character in the last month
	GetByCharacterIdLastMonth(characterId uint32) ([]Model, error)
	// ByCharacterIdLastMonthProvider returns a provider for fame logs for a character in the last month
	ByCharacterIdLastMonthProvider(characterId uint32) model.Provider[[]Model]

	// RequestChange requests a fame change
	RequestChange(mb *message.Buffer) func(transactionId uuid.UUID) func(worldId world.Id) func(channelId channel.Id) func(characterId uint32) func(mapId uint32) func(targetId uint32) func(amount int8) error
	// RequestChangeAndEmit requests a fame change and emits a message
	RequestChangeAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, characterId uint32, mapId uint32, targetId uint32, amount int8) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

func (p *ProcessorImpl) ByCharacterIdLastMonthProvider(characterId uint32) model.Provider[[]Model] {
	return model.SliceMap(Make)(byCharacterIdLastMonthEntityProvider(p.t.Id(), characterId)(p.db))(model.ParallelMap())
}

func (p *ProcessorImpl) GetByCharacterIdLastMonth(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdLastMonthProvider(characterId)()
}

func (p *ProcessorImpl) RequestChange(mb *message.Buffer) func(transactionId uuid.UUID) func(worldId world.Id) func(channelId channel.Id) func(characterId uint32) func(mapId uint32) func(targetId uint32) func(amount int8) error {
	return func(transactionId uuid.UUID) func(worldId world.Id) func(channelId channel.Id) func(characterId uint32) func(mapId uint32) func(targetId uint32) func(amount int8) error {
		return func(worldId world.Id) func(channelId channel.Id) func(characterId uint32) func(mapId uint32) func(targetId uint32) func(amount int8) error {
			return func(channelId channel.Id) func(characterId uint32) func(mapId uint32) func(targetId uint32) func(amount int8) error {
				return func(characterId uint32) func(mapId uint32) func(targetId uint32) func(amount int8) error {
					return func(mapId uint32) func(targetId uint32) func(amount int8) error {
						return func(targetId uint32) func(amount int8) error {
							return func(amount int8) error {
								return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
									characterProcessor := character.NewProcessor(p.l, p.ctx, tx)
									c, err := characterProcessor.GetById(characterId)
									if err != nil {
										return mb.Put(messageFame.EnvEventTopicFameStatus, errorEventStatusProvider(transactionId, worldId, channelId, characterId, messageFame.StatusEventErrorTypeUnexpected))
									}

									_, err = characterProcessor.GetById(targetId)
									if err != nil {
										return mb.Put(messageFame.EnvEventTopicFameStatus, errorEventStatusProvider(transactionId, worldId, channelId, characterId, messageFame.StatusEventErrorInvalidName))
									}

									if c.Level() < 15 {
										return mb.Put(messageFame.EnvEventTopicFameStatus, errorEventStatusProvider(transactionId, worldId, channelId, characterId, messageFame.StatusEventErrorTypeNotMinimumLevel))
									}

									fls, err := p.GetByCharacterIdLastMonth(characterId)
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
										return mb.Put(messageFame.EnvEventTopicFameStatus, errorEventStatusProvider(transactionId, worldId, channelId, characterId, messageFame.StatusEventErrorTypeNotToday))
									}
									if famedTargetLastMonth {
										return mb.Put(messageFame.EnvEventTopicFameStatus, errorEventStatusProvider(transactionId, worldId, channelId, characterId, messageFame.StatusEventErrorTypeNotThisMonth))
									}

									_, err = create(tx, p.t, characterId, targetId, amount)
									if err != nil {
										return mb.Put(messageFame.EnvEventTopicFameStatus, errorEventStatusProvider(transactionId, worldId, channelId, characterId, messageFame.StatusEventErrorTypeUnexpected))
									}

									return characterProcessor.RequestChangeFame(mb)(transactionId)(targetId)(worldId)(characterId)(amount)
								})
							}
						}
					}
				}
			}
		}
	}
}

func (p *ProcessorImpl) RequestChangeAndEmit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, characterId uint32, mapId uint32, targetId uint32, amount int8) error {
	producerProvider := producer.ProviderImpl(p.l)(p.ctx)
	return message.Emit(producerProvider)(func(mb *message.Buffer) error {
		return p.RequestChange(mb)(transactionId)(worldId)(channelId)(characterId)(mapId)(targetId)(amount)
	})
}

// Legacy functions for backward compatibility

func byCharacterIdLastMonthProvider(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
			return func(characterId uint32) model.Provider[[]Model] {
				return model.SliceMap(Make)(byCharacterIdLastMonthEntityProvider(t.Id(), characterId)(db))(model.ParallelMap())
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
		return func(db *gorm.DB) func(worldId byte, channelId byte, characterId uint32, mapId uint32, targetId uint32, amount int8) error {
			return func(worldId byte, channelId byte, characterId uint32, mapId uint32, targetId uint32, amount int8) error {
				processor := NewProcessor(l, ctx, db)
				return processor.RequestChangeAndEmit(uuid.New(), world.Id(worldId), channel.Id(channelId), characterId, mapId, targetId, amount)
			}
		}
	}
}
