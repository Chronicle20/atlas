package invite

import (
	"atlas-saga-orchestrator/kafka/message/invite"
	"atlas-saga-orchestrator/kafka/producer"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Create(transactionId uuid.UUID, inviteType string, actorId uint32, worldId byte, referenceId uint32, targetId uint32) error
	Accept(transactionId uuid.UUID, inviteType string, worldId byte, referenceId uint32, targetId uint32) error
	Reject(transactionId uuid.UUID, inviteType string, worldId byte, originatorId uint32, targetId uint32) error
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

func (p *ProcessorImpl) Create(transactionId uuid.UUID, inviteType string, actorId uint32, worldId byte, referenceId uint32, targetId uint32) error {
	p.l.WithFields(logrus.Fields{
		"transaction_id": transactionId.String(),
		"invite_type":    inviteType,
		"originator_id":  actorId,
		"reference_id":   referenceId,
		"target_id":      targetId,
	}).Debug("Creating invitation.")
	return producer.ProviderImpl(p.l)(p.ctx)(invite.EnvCommandTopic)(createInviteCommandProvider(transactionId, inviteType, actorId, referenceId, worldId, targetId))
}

func (p *ProcessorImpl) Accept(transactionId uuid.UUID, inviteType string, worldId byte, referenceId uint32, targetId uint32) error {
	p.l.WithFields(logrus.Fields{
		"transaction_id": transactionId.String(),
		"invite_type":    inviteType,
		"reference_id":   referenceId,
		"target_id":      targetId,
	}).Debug("Accepting invitation.")
	return producer.ProviderImpl(p.l)(p.ctx)(invite.EnvCommandTopic)(acceptInviteCommandProvider(transactionId, inviteType, worldId, referenceId, targetId))
}

func (p *ProcessorImpl) Reject(transactionId uuid.UUID, inviteType string, worldId byte, originatorId uint32, targetId uint32) error {
	p.l.WithFields(logrus.Fields{
		"transaction_id": transactionId.String(),
		"invite_type":    inviteType,
		"originator_id":  originatorId,
		"target_id":      targetId,
	}).Debug("Rejecting invitation.")
	return producer.ProviderImpl(p.l)(p.ctx)(invite.EnvCommandTopic)(rejectInviteCommandProvider(transactionId, inviteType, worldId, originatorId, targetId))
}
