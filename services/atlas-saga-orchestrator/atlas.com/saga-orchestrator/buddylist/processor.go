package buddylist

import (
	"atlas-saga-orchestrator/kafka/message"
	"context"

	buddylist2 "atlas-saga-orchestrator/kafka/message/buddylist"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	IncreaseCapacityAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, newCapacity byte) error
	IncreaseCapacity(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, worldId world.Id, newCapacity byte) error
	// RequestDeleteAndEmit severs characterId's buddy relationship with
	// targetId (one direction only). Used twice per pair by the
	// world-transfer saga (task-227) to sever both directions.
	RequestDeleteAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error
	RequestDelete(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error
	// RestoreAndEmit puts targetId back on characterId's buddy list (one
	// direction only), the exact inverse of RequestDeleteAndEmit. Used twice
	// per pair by the world-transfer saga's compensation (task-227 FR-4.8) to
	// undo both directions of a severance the player never asked for and
	// cannot re-establish while the buddy is offline.
	RestoreAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error
	Restore(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	p   producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) IncreaseCapacityAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, newCapacity byte) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.IncreaseCapacity(mb)(transactionId, characterId, worldId, newCapacity)
	})
}

func (p *ProcessorImpl) IncreaseCapacity(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, worldId world.Id, newCapacity byte) error {
	return func(transactionId uuid.UUID, characterId uint32, worldId world.Id, newCapacity byte) error {
		return mb.Put(buddylist2.EnvCommandTopic, IncreaseCapacityProvider(transactionId, character.Id(characterId), worldId, newCapacity))
	}
}

func (p *ProcessorImpl) RequestDeleteAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.RequestDelete(mb)(transactionId, characterId, worldId, targetId)
	})
}

func (p *ProcessorImpl) RequestDelete(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error {
		return mb.Put(buddylist2.EnvCommandTopic, RequestDeleteProvider(transactionId, character.Id(characterId), worldId, character.Id(targetId)))
	}
}

func (p *ProcessorImpl) RestoreAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.Restore(mb)(transactionId, characterId, worldId, targetId)
	})
}

func (p *ProcessorImpl) Restore(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, worldId world.Id, targetId uint32) error {
		return mb.Put(buddylist2.EnvCommandTopic, RestoreProvider(transactionId, character.Id(characterId), worldId, character.Id(targetId)))
	}
}
