package character

import (
	"atlas-fame/kafka/message"
	messageCharacter "atlas-fame/kafka/message/character"
	"atlas-fame/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	// GetById gets a character by ID
	GetById(characterId uint32) (Model, error)
	// ByIdProvider returns a provider for a character by ID
	ByIdProvider(characterId uint32) model.Provider[Model]

	// RequestChangeFame requests a fame change
	RequestChangeFame(mb *message.Buffer) func(transactionId uuid.UUID) func(characterId uint32) func(worldId world.Id) func(actorId uint32) func(amount int8) error
	// RequestChangeFameAndEmit requests a fame change and emits a message
	RequestChangeFameAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, actorId uint32, amount int8) error
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

func (p *ProcessorImpl) ByIdProvider(characterId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)
}

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	return p.ByIdProvider(characterId)()
}

func (p *ProcessorImpl) RequestChangeFame(mb *message.Buffer) func(transactionId uuid.UUID) func(characterId uint32) func(worldId world.Id) func(actorId uint32) func(amount int8) error {
	return func(transactionId uuid.UUID) func(characterId uint32) func(worldId world.Id) func(actorId uint32) func(amount int8) error {
		return func(characterId uint32) func(worldId world.Id) func(actorId uint32) func(amount int8) error {
			return func(worldId world.Id) func(actorId uint32) func(amount int8) error {
				return func(actorId uint32) func(amount int8) error {
					return func(amount int8) error {
						mp := requestChangeFameCommandProvider(transactionId, characterId, worldId, actorId, amount)
						return mb.Put(messageCharacter.EnvCommandTopic, mp)
					}
				}
			}
		}
	}
}

func (p *ProcessorImpl) RequestChangeFameAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, actorId uint32, amount int8) error {
	producerProvider := producer.ProviderImpl(p.l)(p.ctx)
	return message.Emit(producerProvider)(func(mb *message.Buffer) error {
		return p.RequestChangeFame(mb)(transactionId)(characterId)(worldId)(actorId)(amount)
	})
}
