package character

import (
	"atlas-buffs/buff/stat"
	character2 "atlas-buffs/kafka/message/character"
	"atlas-buffs/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(characterId uint32) (Model, error)
	Apply(worldId byte, characterId uint32, fromId uint32, sourceId int32, duration int32, changes []stat.Model) error
	Cancel(worldId byte, characterId uint32, sourceId int32) error
	ExpireBuffs() error
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

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	return GetRegistry().Get(p.t, characterId)
}

func (p *ProcessorImpl) Apply(worldId byte, characterId uint32, fromId uint32, sourceId int32, duration int32, changes []stat.Model) error {
	b := GetRegistry().Apply(p.t, worldId, characterId, sourceId, duration, changes)
	_ = producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventStatusTopic)(appliedStatusEventProvider(worldId, characterId, fromId, sourceId, duration, changes, b.CreatedAt(), b.ExpiresAt()))
	return nil
}

func (p *ProcessorImpl) Cancel(worldId byte, characterId uint32, sourceId int32) error {
	b, err := GetRegistry().Cancel(p.t, characterId, sourceId)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	_ = producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventStatusTopic)(expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt()))
	return nil
}

func (p *ProcessorImpl) ExpireBuffs() error {
	for _, c := range GetRegistry().GetCharacters(p.t) {
		ebs := GetRegistry().GetExpired(p.t, c.Id())
		for _, eb := range ebs {
			p.l.Debugf("Expired buff for character [%d] from [%d].", c.Id(), eb.SourceId())
			_ = producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventStatusTopic)(expiredStatusEventProvider(c.WorldId(), c.Id(), eb.SourceId(), eb.Duration(), eb.Changes(), eb.CreatedAt(), eb.ExpiresAt()))
		}
	}
	return nil
}

func ExpireBuffs(l logrus.FieldLogger, ctx context.Context) error {
	ts, err := GetRegistry().GetTenants()
	if err != nil {
		return err
	}

	for _, t := range ts {
		go func() {
			tctx := tenant.WithContext(ctx, t)
			_ = NewProcessor(l, tctx).ExpireBuffs()
		}()
	}
	return nil
}
