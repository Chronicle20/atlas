package character

import (
	"atlas-transports/data/portal"
	"atlas-transports/kafka/message"
	character2 "atlas-transports/kafka/message/character"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	WarpRandom(mb *message.Buffer) func(characterId uint32) func(fieldId field.Id) error
	WarpToPortal(mb *message.Buffer) func(characterId uint32, fieldId field.Id, pp model.Provider[uint32]) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	pp  portal.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		pp:  portal.NewProcessor(l, ctx),
	}
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
