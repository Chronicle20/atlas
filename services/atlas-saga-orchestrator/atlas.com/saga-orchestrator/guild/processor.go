package guild

import (
	"atlas-saga-orchestrator/kafka/message/guild"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
)

type Processor interface {
	RequestName(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
	RequestEmblem(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
	RequestDisband(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
	RequestCapacityIncrease(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
	// RequestLeave emits a LEAVE command for characterId against guildId. force
	// bypasses any client-side confirmation the guild service would otherwise
	// require — used by the world-transfer saga (task-227), which must
	// guarantee the severance rather than wait on player confirmation.
	RequestLeave(transactionId uuid.UUID, characterId uint32, guildId uint32, force bool) error
	// RequestRejoin emits a REJOIN command that puts characterId back in
	// guildId at title. It is the inverse of RequestLeave and exists solely
	// for the world-transfer saga's compensation (task-227 FR-4.8): the
	// forced leave is not something the player can undo themselves, so the
	// rank has to be restored exactly, from the payload the leave step
	// carried.
	RequestRejoin(transactionId uuid.UUID, characterId uint32, guildId uint32, title byte) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) RequestName(transactionId uuid.UUID, ch channel.Model, characterId uint32) error {
	p.l.Debugf("Requesting character [%d] input guild name for creation.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(guild.EnvCommandTopic)(RequestNameProvider(transactionId, ch, characterId))
}

func (p *ProcessorImpl) RequestEmblem(transactionId uuid.UUID, ch channel.Model, characterId uint32) error {
	p.l.Debugf("Requesting character [%d] input new guild emblem.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(guild.EnvCommandTopic)(RequestEmblemProvider(transactionId, ch, characterId))
}

func (p *ProcessorImpl) RequestDisband(transactionId uuid.UUID, ch channel.Model, characterId uint32) error {
	p.l.Debugf("Character [%d] attempting to disband guild.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(guild.EnvCommandTopic)(RequestDisbandProvider(transactionId, ch, characterId))
}

func (p *ProcessorImpl) RequestCapacityIncrease(transactionId uuid.UUID, ch channel.Model, characterId uint32) error {
	p.l.Debugf("Character [%d] attempting to increase guild capacity.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(guild.EnvCommandTopic)(RequestCapacityIncreaseProvider(transactionId, ch, characterId))
}

func (p *ProcessorImpl) RequestLeave(transactionId uuid.UUID, characterId uint32, guildId uint32, force bool) error {
	p.l.Debugf("Character [%d] leaving guild [%d]. Forced? [%t].", characterId, guildId, force)
	return producer.ProviderImpl(p.l)(p.ctx)(guild.EnvCommandTopic)(RequestLeaveProvider(transactionId, characterId, guildId, force))
}

func (p *ProcessorImpl) RequestRejoin(transactionId uuid.UUID, characterId uint32, guildId uint32, title byte) error {
	p.l.Debugf("Character [%d] rejoining guild [%d] at title [%d].", characterId, guildId, title)
	return producer.ProviderImpl(p.l)(p.ctx)(guild.EnvCommandTopic)(RequestRejoinProvider(transactionId, characterId, guildId, title))
}
