package invite

import (
	invite2 "atlas-invites/kafka/message/invite"
	"atlas-invites/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const StartInviteId = uint32(1000000000)

type Processor interface {
	GetByCharacterId(characterId uint32) ([]Model, error)
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	Create(referenceId uint32, worldId byte, inviteType string, originatorId uint32, targetId uint32) error
	Accept(referenceId uint32, worldId byte, inviteType string, actorId uint32) error
	Reject(originatorId uint32, worldId byte, inviteType string, actorId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
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

func (p *ProcessorImpl) Create(referenceId uint32, worldId byte, inviteType string, originatorId uint32, targetId uint32) error {
	i := GetRegistry().Create(p.t, originatorId, worldId, targetId, inviteType, referenceId)
	return producer.ProviderImpl(p.l)(p.ctx)(invite2.EnvEventStatusTopic)(createdStatusEventProvider(i.ReferenceId(), worldId, inviteType, i.OriginatorId(), i.TargetId()))
}

func (p *ProcessorImpl) Accept(referenceId uint32, worldId byte, inviteType string, actorId uint32) error {
	i, err := GetRegistry().GetByReference(p.t, actorId, inviteType, referenceId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate invite being acted upon.")
		return err
	}

	err = GetRegistry().Delete(p.t, actorId, inviteType, i.OriginatorId())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate invite being acted upon.")
		return err
	}

	return producer.ProviderImpl(p.l)(p.ctx)(invite2.EnvEventStatusTopic)(acceptedStatusEventProvider(i.ReferenceId(), worldId, inviteType, i.OriginatorId(), i.TargetId()))
}

func (p *ProcessorImpl) Reject(originatorId uint32, worldId byte, inviteType string, actorId uint32) error {
	i, err := GetRegistry().GetByOriginator(p.t, actorId, inviteType, originatorId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate invite being acted upon.")
		return err
	}

	err = GetRegistry().Delete(p.t, actorId, inviteType, originatorId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate invite being acted upon.")
		return err
	}

	return producer.ProviderImpl(p.l)(p.ctx)(invite2.EnvEventStatusTopic)(rejectedStatusEventProvider(i.ReferenceId(), worldId, inviteType, i.OriginatorId(), i.TargetId()))
}
