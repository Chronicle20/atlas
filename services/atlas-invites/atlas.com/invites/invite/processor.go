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
							i := GetRegistry().Create(p.t, originatorId, worldId, targetId, inviteType, referenceId)
							err := mb.Put(invite2.EnvEventStatusTopic, createdStatusEventProvider(i.ReferenceId(), worldId, inviteType, i.OriginatorId(), i.TargetId(), transactionId))
							if err != nil {
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
						i, err := GetRegistry().GetByReference(p.t, actorId, inviteType, referenceId)
						if err != nil {
							p.l.WithError(err).Errorf("Unable to locate invite being acted upon.")
							return Model{}, err
						}

						err = GetRegistry().Delete(p.t, actorId, inviteType, i.OriginatorId())
						if err != nil {
							p.l.WithError(err).Errorf("Unable to locate invite being acted upon.")
							return Model{}, err
						}

						err = mb.Put(invite2.EnvEventStatusTopic, acceptedStatusEventProvider(i.ReferenceId(), worldId, inviteType, i.OriginatorId(), i.TargetId(), transactionId))
						if err != nil {
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
						i, err := GetRegistry().GetByOriginator(p.t, actorId, inviteType, originatorId)
						if err != nil {
							p.l.WithError(err).Errorf("Unable to locate invite being acted upon.")
							return Model{}, err
						}

						err = GetRegistry().Delete(p.t, actorId, inviteType, originatorId)
						if err != nil {
							p.l.WithError(err).Errorf("Unable to locate invite being acted upon.")
							return Model{}, err
						}

						err = mb.Put(invite2.EnvEventStatusTopic, rejectedStatusEventProvider(i.ReferenceId(), worldId, inviteType, i.OriginatorId(), i.TargetId(), transactionId))
						if err != nil {
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
