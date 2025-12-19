package invite

import (
	"atlas-invites/kafka/message"
	invite2 "atlas-invites/kafka/message/invite"
	"atlas-invites/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const StartInviteId = uint32(1000000000)

type Processor interface {
	GetByCharacterId(characterId uint32) ([]Model, error)
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	CreateAndEmit(referenceId uint32, worldId byte, inviteType string, originatorId uint32, targetId uint32, transactionId uuid.UUID) (Model, error)
	Create(mb *message.Buffer) func(referenceId uint32) func(worldId byte) func(inviteType string) func(originatorId uint32) func(targetId uint32) func(transactionId uuid.UUID) (Model, error)
	AcceptAndEmit(referenceId uint32, worldId byte, inviteType string, actorId uint32, transactionId uuid.UUID) (Model, error)
	Accept(mb *message.Buffer) func(referenceId uint32) func(worldId byte) func(inviteType string) func(actorId uint32) func(transactionId uuid.UUID) (Model, error)
	RejectAndEmit(originatorId uint32, worldId byte, inviteType string, actorId uint32, transactionId uuid.UUID) (Model, error)
	Reject(mb *message.Buffer) func(originatorId uint32) func(worldId byte) func(inviteType string) func(actorId uint32) func(transactionId uuid.UUID) (Model, error)
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

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	is, err := GetRegistry().GetForCharacter(p.t, characterId)
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}
	return model.FixedProvider(is)
}

// Create implements the business logic for creating an invite
func (p *ProcessorImpl) Create(mb *message.Buffer) func(referenceId uint32) func(worldId byte) func(inviteType string) func(originatorId uint32) func(targetId uint32) func(transactionId uuid.UUID) (Model, error) {
	return func(referenceId uint32) func(worldId byte) func(inviteType string) func(originatorId uint32) func(targetId uint32) func(transactionId uuid.UUID) (Model, error) {
		return func(worldId byte) func(inviteType string) func(originatorId uint32) func(targetId uint32) func(transactionId uuid.UUID) (Model, error) {
			return func(inviteType string) func(originatorId uint32) func(targetId uint32) func(transactionId uuid.UUID) (Model, error) {
				return func(originatorId uint32) func(targetId uint32) func(transactionId uuid.UUID) (Model, error) {
					return func(targetId uint32) func(transactionId uuid.UUID) (Model, error) {
						return func(transactionId uuid.UUID) (Model, error) {
							p.l.WithFields(logrus.Fields{
								"referenceId":  referenceId,
								"worldId":      worldId,
								"inviteType":   inviteType,
								"originatorId": originatorId,
								"targetId":     targetId,
								"transaction":  transactionId.String(),
							}).Debug("Creating invite")

							i := GetRegistry().Create(p.t, originatorId, worldId, targetId, inviteType, referenceId)

							p.l.WithFields(logrus.Fields{
								"inviteId":     i.Id(),
								"referenceId":  i.ReferenceId(),
								"worldId":      i.WorldId(),
								"inviteType":   i.Type(),
								"originatorId": i.OriginatorId(),
								"targetId":     i.TargetId(),
								"transaction":  transactionId.String(),
							}).Info("Invite created successfully")

							err := mb.Put(invite2.EnvEventStatusTopic, createdStatusEventProvider(i.ReferenceId(), worldId, inviteType, i.OriginatorId(), i.TargetId(), transactionId))
							if err != nil {
								p.l.WithError(err).WithFields(logrus.Fields{
									"inviteId":     i.Id(),
									"referenceId":  i.ReferenceId(),
									"transaction":  transactionId.String(),
								}).Error("Failed to put created event in message buffer")
								return Model{}, err
							}
							return i, nil
						}
					}
				}
			}
		}
	}
}

// CreateAndEmit implements the business logic for creating an invite and emitting the event
func (p *ProcessorImpl) CreateAndEmit(referenceId uint32, worldId byte, inviteType string, originatorId uint32, targetId uint32, transactionId uuid.UUID) (Model, error) {
	var m Model
	err := message.Emit(p.p)(func(buf *message.Buffer) error {
		var err error
		m, err = p.Create(buf)(referenceId)(worldId)(inviteType)(originatorId)(targetId)(transactionId)
		return err
	})
	return m, err
}

// Accept implements the business logic for accepting an invite
func (p *ProcessorImpl) Accept(mb *message.Buffer) func(referenceId uint32) func(worldId byte) func(inviteType string) func(actorId uint32) func(transactionId uuid.UUID) (Model, error) {
	return func(referenceId uint32) func(worldId byte) func(inviteType string) func(actorId uint32) func(transactionId uuid.UUID) (Model, error) {
		return func(worldId byte) func(inviteType string) func(actorId uint32) func(transactionId uuid.UUID) (Model, error) {
			return func(inviteType string) func(actorId uint32) func(transactionId uuid.UUID) (Model, error) {
				return func(actorId uint32) func(transactionId uuid.UUID) (Model, error) {
					return func(transactionId uuid.UUID) (Model, error) {
						p.l.WithFields(logrus.Fields{
							"referenceId": referenceId,
							"worldId":     worldId,
							"inviteType":  inviteType,
							"actorId":     actorId,
							"transaction": transactionId.String(),
						}).Debug("Accepting invite")

						i, err := GetRegistry().GetByReference(p.t, actorId, inviteType, referenceId)
						if err != nil {
							p.l.WithError(err).WithFields(logrus.Fields{
								"referenceId": referenceId,
								"inviteType":  inviteType,
								"actorId":     actorId,
								"transaction": transactionId.String(),
							}).Error("Unable to locate invite being acted upon")
							return Model{}, err
						}

						p.l.WithFields(logrus.Fields{
							"inviteId":     i.Id(),
							"referenceId":  i.ReferenceId(),
							"inviteType":   i.Type(),
							"originatorId": i.OriginatorId(),
							"targetId":     i.TargetId(),
							"transaction":  transactionId.String(),
						}).Debug("Found invite to accept")

						err = GetRegistry().Delete(p.t, actorId, inviteType, i.OriginatorId())
						if err != nil {
							p.l.WithError(err).WithFields(logrus.Fields{
								"inviteId":     i.Id(),
								"referenceId":  i.ReferenceId(),
								"inviteType":   i.Type(),
								"originatorId": i.OriginatorId(),
								"actorId":      actorId,
								"transaction":  transactionId.String(),
							}).Error("Unable to delete invite being accepted")
							return Model{}, err
						}

						p.l.WithFields(logrus.Fields{
							"inviteId":     i.Id(),
							"referenceId":  i.ReferenceId(),
							"inviteType":   i.Type(),
							"originatorId": i.OriginatorId(),
							"targetId":     i.TargetId(),
							"transaction":  transactionId.String(),
						}).Info("Invite accepted successfully")

						err = mb.Put(invite2.EnvEventStatusTopic, acceptedStatusEventProvider(i.ReferenceId(), worldId, inviteType, i.OriginatorId(), i.TargetId(), transactionId))
						if err != nil {
							p.l.WithError(err).WithFields(logrus.Fields{
								"inviteId":     i.Id(),
								"referenceId":  i.ReferenceId(),
								"transaction":  transactionId.String(),
							}).Error("Failed to put accepted event in message buffer")
							return Model{}, err
						}
						return i, nil
					}
				}
			}
		}
	}
}

// AcceptAndEmit implements the business logic for accepting an invite and emitting the event
func (p *ProcessorImpl) AcceptAndEmit(referenceId uint32, worldId byte, inviteType string, actorId uint32, transactionId uuid.UUID) (Model, error) {
	var m Model
	err := message.Emit(p.p)(func(buf *message.Buffer) error {
		var err error
		m, err = p.Accept(buf)(referenceId)(worldId)(inviteType)(actorId)(transactionId)
		return err
	})
	return m, err
}

// Reject implements the business logic for rejecting an invite
func (p *ProcessorImpl) Reject(mb *message.Buffer) func(originatorId uint32) func(worldId byte) func(inviteType string) func(actorId uint32) func(transactionId uuid.UUID) (Model, error) {
	return func(originatorId uint32) func(worldId byte) func(inviteType string) func(actorId uint32) func(transactionId uuid.UUID) (Model, error) {
		return func(worldId byte) func(inviteType string) func(actorId uint32) func(transactionId uuid.UUID) (Model, error) {
			return func(inviteType string) func(actorId uint32) func(transactionId uuid.UUID) (Model, error) {
				return func(actorId uint32) func(transactionId uuid.UUID) (Model, error) {
					return func(transactionId uuid.UUID) (Model, error) {
						p.l.WithFields(logrus.Fields{
							"originatorId": originatorId,
							"worldId":      worldId,
							"inviteType":   inviteType,
							"actorId":      actorId,
							"transaction":  transactionId.String(),
						}).Debug("Rejecting invite")

						i, err := GetRegistry().GetByOriginator(p.t, actorId, inviteType, originatorId)
						if err != nil {
							p.l.WithError(err).WithFields(logrus.Fields{
								"originatorId": originatorId,
								"inviteType":   inviteType,
								"actorId":      actorId,
								"transaction":  transactionId.String(),
							}).Error("Unable to locate invite being acted upon")
							return Model{}, err
						}

						p.l.WithFields(logrus.Fields{
							"inviteId":     i.Id(),
							"referenceId":  i.ReferenceId(),
							"inviteType":   i.Type(),
							"originatorId": i.OriginatorId(),
							"targetId":     i.TargetId(),
							"transaction":  transactionId.String(),
						}).Debug("Found invite to reject")

						err = GetRegistry().Delete(p.t, actorId, inviteType, originatorId)
						if err != nil {
							p.l.WithError(err).WithFields(logrus.Fields{
								"inviteId":     i.Id(),
								"referenceId":  i.ReferenceId(),
								"inviteType":   i.Type(),
								"originatorId": i.OriginatorId(),
								"actorId":      actorId,
								"transaction":  transactionId.String(),
							}).Error("Unable to delete invite being rejected")
							return Model{}, err
						}

						p.l.WithFields(logrus.Fields{
							"inviteId":     i.Id(),
							"referenceId":  i.ReferenceId(),
							"inviteType":   i.Type(),
							"originatorId": i.OriginatorId(),
							"targetId":     i.TargetId(),
							"transaction":  transactionId.String(),
						}).Info("Invite rejected successfully")

						err = mb.Put(invite2.EnvEventStatusTopic, rejectedStatusEventProvider(i.ReferenceId(), worldId, inviteType, i.OriginatorId(), i.TargetId(), transactionId))
						if err != nil {
							p.l.WithError(err).WithFields(logrus.Fields{
								"inviteId":     i.Id(),
								"referenceId":  i.ReferenceId(),
								"transaction":  transactionId.String(),
							}).Error("Failed to put rejected event in message buffer")
							return Model{}, err
						}
						return i, nil
					}
				}
			}
		}
	}
}

// RejectAndEmit implements the business logic for rejecting an invite and emitting the event
func (p *ProcessorImpl) RejectAndEmit(originatorId uint32, worldId byte, inviteType string, actorId uint32, transactionId uuid.UUID) (Model, error) {
	var m Model
	err := message.Emit(p.p)(func(buf *message.Buffer) error {
		var err error
		m, err = p.Reject(buf)(originatorId)(worldId)(inviteType)(actorId)(transactionId)
		return err
	})
	return m, err
}
