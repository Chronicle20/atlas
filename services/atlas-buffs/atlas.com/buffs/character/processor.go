package character

import (
	"atlas-buffs/buff/stat"
	"atlas-buffs/kafka/message"
	character2 "atlas-buffs/kafka/message/character"
	"context"
	"errors"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(characterId uint32) (Model, error)
	Apply(worldId world.Id, characterId uint32, fromId uint32, sourceId int32, duration int32, changes []stat.Model) error
	Cancel(worldId world.Id, characterId uint32, sourceId int32) error
	CancelAll(worldId world.Id, characterId uint32) error
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

func (p *ProcessorImpl) Apply(worldId world.Id, characterId uint32, fromId uint32, sourceId int32, duration int32, changes []stat.Model) error {
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		b, err := GetRegistry().Apply(p.t, worldId, characterId, sourceId, duration, changes)
		if err != nil {
			return err
		}
		return buf.Put(character2.EnvEventStatusTopic, appliedStatusEventProvider(worldId, characterId, fromId, sourceId, duration, changes, b.CreatedAt(), b.ExpiresAt()))
	})
}

func (p *ProcessorImpl) Cancel(worldId world.Id, characterId uint32, sourceId int32) error {
	b, err := GetRegistry().Cancel(p.t, characterId, sourceId)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		return buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt()))
	})
}

func (p *ProcessorImpl) CancelAll(worldId world.Id, characterId uint32) error {
	buffs := GetRegistry().CancelAll(p.t, characterId)
	if len(buffs) == 0 {
		return nil
	}
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, b := range buffs {
			if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt())); err != nil {
				return err
			}
		}
		return nil
	})
}

func (p *ProcessorImpl) ExpireBuffs() error {
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, c := range GetRegistry().GetCharacters(p.t) {
			ebs := GetRegistry().GetExpired(p.t, c.Id())
			for _, eb := range ebs {
				p.l.Debugf("Expired buff for character [%d] from [%d].", c.Id(), eb.SourceId())
				if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(c.WorldId(), c.Id(), eb.SourceId(), eb.Duration(), eb.Changes(), eb.CreatedAt(), eb.ExpiresAt())); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func ExpireBuffs(l logrus.FieldLogger, ctx context.Context) error {
	ts, err := GetRegistry().GetTenants()
	if err != nil {
		return err
	}

	for _, t := range ts {
		go func() {
			tctx := tenant.WithContext(ctx, t)
			if err := NewProcessor(l, tctx).ExpireBuffs(); err != nil {
				l.WithError(err).Error("Failed to expire buffs for tenant.")
			}
		}()
	}
	return nil
}
