package fame

import (
	"atlas-fame/character"
	"atlas-fame/kafka/message"
	messageFame "atlas-fame/kafka/message/fame"
	"atlas-fame/kafka/producer"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// errFameChangeRejected is an internal sentinel used to abort the
// RequestChange transaction closure on a handled validation rejection
// (character not found, below minimum level, already famed today/this
// month, etc.) whose status event must fire on the direct producer path
// rather than ride into the outbox alongside a state change that never
// committed (recipe failure-path pitfall #1). It never escapes
// RequestChange(): the rejectEmit != nil check short-circuits it.
var errFameChangeRejected = errors.New("fame change rejected")

type Processor interface {
	// WithTransaction returns a copy of this processor bound to the given transaction
	WithTransaction(tx *gorm.DB) Processor

	// GetByCharacterIdLastMonth gets all fame logs for a character in the last month
	GetByCharacterIdLastMonth(characterId uint32) ([]Model, error)
	// ByCharacterIdLastMonthProvider returns a provider for fame logs for a character in the last month
	ByCharacterIdLastMonthProvider(characterId uint32) model.Provider[[]Model]

	// RequestChange requests a fame change
	RequestChange(mb *message.Buffer) func(transactionId uuid.UUID) func(field field.Model) func(characterId uint32) func(targetId uint32) func(amount int8) error
	// RequestChangeAndEmit requests a fame change and emits a message
	RequestChangeAndEmit(transactionId uuid.UUID, field field.Model, characterId uint32, targetId uint32, amount int8) error

	// DeleteByCharacterId deletes all fame logs involving a character (as giver or receiver)
	DeleteByCharacterId(characterId uint32) error
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{l: p.l, ctx: p.ctx, db: tx, t: p.t}
}

func (p *ProcessorImpl) ByCharacterIdLastMonthProvider(characterId uint32) model.Provider[[]Model] {
	return model.SliceMap(Make)(byCharacterIdLastMonthEntityProvider(characterId)(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

func (p *ProcessorImpl) GetByCharacterIdLastMonth(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdLastMonthProvider(characterId)()
}

func (p *ProcessorImpl) RequestChange(mb *message.Buffer) func(transactionId uuid.UUID) func(field field.Model) func(characterId uint32) func(targetId uint32) func(amount int8) error {
	return func(transactionId uuid.UUID) func(field field.Model) func(characterId uint32) func(targetId uint32) func(amount int8) error {
		return func(field field.Model) func(characterId uint32) func(targetId uint32) func(amount int8) error {
			return func(characterId uint32) func(targetId uint32) func(amount int8) error {
				return func(targetId uint32) func(amount int8) error {
					return func(amount int8) error {
						// rejectEmit captures a handled validation rejection (no state
						// change committed) so it can be fired on the direct producer
						// path, outside the outbox-bound tx, instead of leaking into
						// the outbox as if it were part of the committed transaction.
						var rejectEmit func() error
						txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
							characterProcessor := character.NewProcessor(p.l, p.ctx, tx)
							c, err := characterProcessor.GetById(characterId)
							if err != nil {
								rejectEmit = func() error {
									return producer.ProviderImpl(p.l)(p.ctx)(messageFame.EnvEventTopicFameStatus)(errorEventStatusProvider(transactionId, field.Channel(), characterId, messageFame.StatusEventErrorTypeUnexpected))
								}
								return errFameChangeRejected
							}

							_, err = characterProcessor.GetById(targetId)
							if err != nil {
								rejectEmit = func() error {
									return producer.ProviderImpl(p.l)(p.ctx)(messageFame.EnvEventTopicFameStatus)(errorEventStatusProvider(transactionId, field.Channel(), characterId, messageFame.StatusEventErrorInvalidName))
								}
								return errFameChangeRejected
							}

							if c.Level() < 15 {
								rejectEmit = func() error {
									return producer.ProviderImpl(p.l)(p.ctx)(messageFame.EnvEventTopicFameStatus)(errorEventStatusProvider(transactionId, field.Channel(), characterId, messageFame.StatusEventErrorTypeNotMinimumLevel))
								}
								return errFameChangeRejected
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
								rejectEmit = func() error {
									return producer.ProviderImpl(p.l)(p.ctx)(messageFame.EnvEventTopicFameStatus)(errorEventStatusProvider(transactionId, field.Channel(), characterId, messageFame.StatusEventErrorTypeNotToday))
								}
								return errFameChangeRejected
							}
							if famedTargetLastMonth {
								rejectEmit = func() error {
									return producer.ProviderImpl(p.l)(p.ctx)(messageFame.EnvEventTopicFameStatus)(errorEventStatusProvider(transactionId, field.Channel(), characterId, messageFame.StatusEventErrorTypeNotThisMonth))
								}
								return errFameChangeRejected
							}

							_, err = create(tx, p.t.Id(), characterId, targetId, amount)
							if err != nil {
								rejectEmit = func() error {
									return producer.ProviderImpl(p.l)(p.ctx)(messageFame.EnvEventTopicFameStatus)(errorEventStatusProvider(transactionId, field.Channel(), characterId, messageFame.StatusEventErrorTypeUnexpected))
								}
								return errFameChangeRejected
							}

							return characterProcessor.RequestChangeFame(mb)(transactionId)(targetId)(field.WorldId())(characterId)(amount)
						})
						if rejectEmit != nil {
							if emitErr := rejectEmit(); emitErr != nil {
								p.l.WithError(emitErr).Errorf("Unable to emit fame change rejection for character [%d].", characterId)
							}
							return nil
						}
						return txErr
					}
				}
			}
		}
	}
}

func (p *ProcessorImpl) RequestChangeAndEmit(transactionId uuid.UUID, field field.Model, characterId uint32, targetId uint32, amount int8) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(mb *message.Buffer) error {
			return p.WithTransaction(tx).RequestChange(mb)(transactionId)(field)(characterId)(targetId)(amount)
		})
	})
}

func (p *ProcessorImpl) DeleteByCharacterId(characterId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return deleteByCharacterId(tx, characterId)
	})
}
