package skill

import (
	"atlas-saga-orchestrator/kafka/message"
	skill2 "atlas-saga-orchestrator/kafka/message/skill"
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	RequestCreateAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error
	RequestCreate(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error
	RequestUpdateAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error
	RequestUpdate(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error
	// RequestDeleteSkill is the saga-compensation dispatch for CreateSkill
	// (plan Phase 5 / Phase 6).
	RequestDeleteSkill(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32) error
	// TransferSPAndEmit emits the TRANSFER_SP command (SP Reset, task-126).
	TransferSPAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, jobId job.Id, fromSkillId uint32, toSkillId uint32, itemTier byte, targetMaxLevel byte) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	p   producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		p:   producer.ProviderImpl(l)(ctx),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) RequestCreateAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.RequestCreate(mb)(transactionId, worldId, characterId, skillId, level, masterLevel, expiration)
	})
}

func (p *ProcessorImpl) RequestCreate(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
		return mb.Put(skill2.EnvCommandTopic, RequestCreateProvider(transactionId, worldId, characterId, skillId, level, masterLevel, expiration))
	}
}

func (p *ProcessorImpl) RequestUpdateAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.RequestUpdate(mb)(transactionId, worldId, characterId, skillId, level, masterLevel, expiration)
	})
}

func (p *ProcessorImpl) RequestUpdate(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32, level byte, masterLevel byte, expiration time.Time) error {
		return mb.Put(skill2.EnvCommandTopic, RequestUpdateProvider(transactionId, worldId, characterId, skillId, level, masterLevel, expiration))
	}
}

// RequestDeleteSkill emits the saga-correlated REQUEST_DELETE command used by
// the character-creation reverse-walk compensator.
func (p *ProcessorImpl) RequestDeleteSkill(transactionId uuid.UUID, worldId world.Id, characterId uint32, skillId uint32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return mb.Put(skill2.EnvCommandTopic, RequestDeleteProvider(transactionId, worldId, characterId, skillId))
	})
}

// TransferSPAndEmit emits the TRANSFER_SP command that moves one skill point
// FromSkillId -> ToSkillId (SP Reset items 5050001-5050004, task-126).
func (p *ProcessorImpl) TransferSPAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, jobId job.Id, fromSkillId uint32, toSkillId uint32, itemTier byte, targetMaxLevel byte) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return mb.Put(skill2.EnvCommandTopic, TransferSpProvider(transactionId, worldId, characterId, jobId, fromSkillId, toSkillId, itemTier, targetMaxLevel))
	})
}
