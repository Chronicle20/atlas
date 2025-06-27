package character

import (
	"atlas-saga-orchestrator/data/portal"
	"atlas-saga-orchestrator/kafka/message"
	character2 "atlas-saga-orchestrator/kafka/message/character"
	"atlas-saga-orchestrator/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	WarpRandom(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error
	WarpRandomAndEmit(characterId uint32, fieldId field.Id) error
	WarpToPortalAndEmit(characterId uint32, fieldId field.Id, pp model.Provider[uint32]) error
	WarpToPortal(mb *message.Buffer) func(characterId uint32, fieldId field.Id, pp model.Provider[uint32]) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	p   producer.Provider
	pp  portal.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
		pp:  portal.NewProcessor(l, ctx),
	}
}

func (p *ProcessorImpl) WarpRandomAndEmit(characterId uint32, fieldId field.Id) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.WarpRandom(mb)(characterId)(fieldId)
	})
}

func (p *ProcessorImpl) WarpRandom(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error {
	return func(characterId uint32) func(fieldId field.Id) error {
		return func(fieldId field.Id) error {
			f, ok := field.FromId(fieldId)
			if !ok {
				return errors.New("invalid field")
			}
			return p.WarpToPortal(mb)(characterId, fieldId, p.pp.RandomSpawnPointIdProvider(f.MapId()))
		}
	}
}

func (p *ProcessorImpl) WarpToPortalAndEmit(characterId uint32, fieldId field.Id, pp model.Provider[uint32]) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.WarpToPortal(mb)(characterId, fieldId, pp)
	})
}

func (p *ProcessorImpl) WarpToPortal(mb *message.Buffer) func(characterId uint32, fieldId field.Id, pp model.Provider[uint32]) error {
	return func(characterId uint32, fieldId field.Id, pp model.Provider[uint32]) error {
		f, ok := field.FromId(fieldId)
		if !ok {
			return errors.New("invalid field")
		}
		portalId, err := pp()
		if err != nil {
			return err
		}
		return mb.Put(character2.EnvCommandTopic, ChangeMapProvider(f.WorldId(), f.ChannelId(), characterId, f.MapId(), portalId))
	}
}
