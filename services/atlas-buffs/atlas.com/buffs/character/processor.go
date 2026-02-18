package character

import (
	"atlas-buffs/buff/stat"
	"atlas-buffs/kafka/message"
	character2 "atlas-buffs/kafka/message/character"
	"context"
	"errors"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(characterId uint32) (Model, error)
	Apply(worldId world.Id, channelId channel.Id, characterId uint32, fromId uint32, sourceId int32, level byte, duration int32, changes []stat.Model) error
	Cancel(worldId world.Id, characterId uint32, sourceId int32) error
	CancelAll(worldId world.Id, characterId uint32) error
	ExpireBuffs() error
	ProcessPoisonTicks() error
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

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	return GetRegistry().Get(p.ctx, characterId)
}

func (p *ProcessorImpl) Apply(worldId world.Id, channelId channel.Id, characterId uint32, fromId uint32, sourceId int32, level byte, duration int32, changes []stat.Model) error {
	if isDiseaseChange(changes) && GetRegistry().HasImmunity(p.ctx, characterId) {
		p.l.Debugf("Character [%d] is immune to disease, skipping apply.", characterId)
		return nil
	}

	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		b, err := GetRegistry().Apply(p.ctx, worldId, channelId, characterId, sourceId, level, duration, changes)
		if err != nil {
			return err
		}
		return buf.Put(character2.EnvEventStatusTopic, appliedStatusEventProvider(worldId, characterId, fromId, sourceId, level, duration, changes, b.CreatedAt(), b.ExpiresAt()))
	})
}

func (p *ProcessorImpl) Cancel(worldId world.Id, characterId uint32, sourceId int32) error {
	b, err := GetRegistry().Cancel(p.ctx, characterId, sourceId)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		return buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Level(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt()))
	})
}

func (p *ProcessorImpl) CancelAll(worldId world.Id, characterId uint32) error {
	buffs := GetRegistry().CancelAll(p.ctx, characterId)
	if len(buffs) == 0 {
		return nil
	}
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, b := range buffs {
			if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Level(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt())); err != nil {
				return err
			}
		}
		return nil
	})
}

func (p *ProcessorImpl) ExpireBuffs() error {
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, c := range GetRegistry().GetCharacters(p.ctx) {
			ebs := GetRegistry().GetExpired(p.ctx, c.Id())
			for _, eb := range ebs {
				p.l.Debugf("Expired buff for character [%d] from [%d].", c.Id(), eb.SourceId())
				if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(c.WorldId(), c.Id(), eb.SourceId(), eb.Level(), eb.Duration(), eb.Changes(), eb.CreatedAt(), eb.ExpiresAt())); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func ExpireBuffs(l logrus.FieldLogger, ctx context.Context) error {
	ts, err := GetRegistry().GetTenants(ctx)
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

func (p *ProcessorImpl) ProcessPoisonTicks() error {
	entries := GetRegistry().GetPoisonCharacters(p.ctx)
	now := time.Now()

	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, entry := range entries {
			lastTick, hasTicked := GetRegistry().GetLastPoisonTick(p.ctx, entry.CharacterId)
			if hasTicked && now.Sub(lastTick) < time.Second {
				continue
			}

			amount := int16(-entry.Amount)
			if amount >= 0 {
				continue
			}

			p.l.Debugf("Poison tick for character [%d], damage [%d].", entry.CharacterId, -amount)

			if err := buf.Put(character2.EnvCommandTopicCharacter, changeHPCommandProvider(entry.WorldId, entry.ChannelId, entry.CharacterId, amount)); err != nil {
				return err
			}

			GetRegistry().UpdatePoisonTick(p.ctx, entry.CharacterId, now)
		}
		return nil
	})
}

func ProcessPoisonTicks(l logrus.FieldLogger, ctx context.Context) error {
	ts, err := GetRegistry().GetTenants(ctx)
	if err != nil {
		return err
	}

	for _, t := range ts {
		go func() {
			tctx := tenant.WithContext(ctx, t)
			if err := NewProcessor(l, tctx).ProcessPoisonTicks(); err != nil {
				l.WithError(err).Error("Failed to process poison ticks for tenant.")
			}
		}()
	}
	return nil
}
