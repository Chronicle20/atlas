package character

import (
	"atlas-buffs/buff/stat"
	"atlas-buffs/kafka/message"
	character2 "atlas-buffs/kafka/message/character"
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	GetById(characterId uint32) (Model, error)
	Apply(worldId world.Id, channelId channel.Id, characterId uint32, fromId uint32, sourceId int32, level byte, duration int32, changes []stat.Model, accumulate bool, noExpiry bool) error
	Cancel(worldId world.Id, characterId uint32, sourceId int32) error
	CancelAll(worldId world.Id, characterId uint32) error
	CancelByStatTypes(worldId world.Id, characterId uint32, types []string) error
	UpdateStatValue(worldId world.Id, channelId channel.Id, characterId uint32, u StatValueUpdate) error
	ExpireBuffs() error
	ExpireForCharacter(worldId world.Id, characterId uint32) error
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	return GetRegistry().Get(p.ctx, characterId)
}

func (p *ProcessorImpl) Apply(worldId world.Id, channelId channel.Id, characterId uint32, fromId uint32, sourceId int32, level byte, duration int32, changes []stat.Model, accumulate bool, noExpiry bool) error {
	if isDiseaseChange(changes) && GetRegistry().HasImmunity(p.ctx, characterId) {
		p.l.Debugf("Character [%d] is immune to disease, skipping apply.", characterId)
		return nil
	}

	err := message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		applied, err := GetRegistry().Apply(p.ctx, worldId, channelId, characterId, sourceId, level, duration, changes, accumulate, noExpiry)
		if err != nil {
			return err
		}
		// One APPLIED per stored buff: default mode returns a single whole-source
		// buff; accumulate mode returns one buff per stat, each carrying its own
		// changes/expiry so the channel sets (and later cancels) each stat icon
		// independently.
		for _, b := range applied {
			if err := buf.Put(character2.EnvEventStatusTopic, appliedStatusEventProvider(worldId, characterId, fromId, b.SourceId(), b.Level(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt(), b.NoExpiry())); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, changes)
	return nil
}

func (p *ProcessorImpl) Cancel(worldId world.Id, characterId uint32, sourceId int32) error {
	cancelled, err := GetRegistry().Cancel(p.ctx, characterId, sourceId)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	// One EXPIRED per removed buff: a sourceId can map to several per-stat buffs
	// in accumulate mode (Beholder Hex), and each needs its own cancel so the
	// client clears every icon rather than leaving the others stuck.
	err = message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, b := range cancelled {
			if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Level(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt(), b.NoExpiry())); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	sets := make([][]stat.Model, 0, len(cancelled))
	for _, b := range cancelled {
		sets = append(sets, b.Changes())
	}
	markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, sets...)
	return nil
}

func (p *ProcessorImpl) CancelAll(worldId world.Id, characterId uint32) error {
	buffs := GetRegistry().CancelAll(p.ctx, characterId)
	if len(buffs) == 0 {
		return nil
	}
	err := message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, b := range buffs {
			if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Level(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt(), b.NoExpiry())); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	sets := make([][]stat.Model, 0, len(buffs))
	for _, b := range buffs {
		sets = append(sets, b.Changes())
	}
	markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, sets...)
	return nil
}

func (p *ProcessorImpl) CancelByStatTypes(worldId world.Id, characterId uint32, types []string) error {
	if len(types) == 0 {
		return nil
	}
	typeSet := make(map[string]bool, len(types))
	for _, t := range types {
		typeSet[t] = true
	}

	cancelled, err := GetRegistry().CancelByStatTypes(p.ctx, characterId, typeSet)
	if err != nil {
		return err
	}
	if len(cancelled) == 0 {
		return nil
	}

	err = message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, b := range cancelled {
			if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Level(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt(), b.NoExpiry())); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	sets := make([][]stat.Model, 0, len(cancelled))
	for _, b := range cancelled {
		sets = append(sets, b.Changes())
	}
	markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, sets...)
	return nil
}

// UpdateStatValue applies a stat-value mutation to an existing buff — or, with
// u.CreateIfMissing, creates the buff — and emits the matching status event: a
// created buff announces APPLIED (it is a new buff, carrying its own
// createdAt/expiresAt and noExpiry flag), a mutated one announces STAT_UPDATED
// with the buff's ORIGINAL timestamps (so the channel re-broadcasts the
// remaining duration and never extends the buff). Missing/expired buff without
// CreateIfMissing, and at-cap increments, stay Debug no-ops — the buff can
// lapse between the channel's attack and this command.
func (p *ProcessorImpl) UpdateStatValue(worldId world.Id, channelId channel.Id, characterId uint32, u StatValueUpdate) error {
	if u.Operation != character2.StatOperationIncrement && u.Operation != character2.StatOperationSet {
		p.l.Warnf("Unknown stat value operation [%s] for character [%d] buff [%d]; ignoring.", u.Operation, characterId, u.SourceId)
		return nil
	}
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		updated, changed, created, err := GetRegistry().UpdateStatValue(p.ctx, worldId, channelId, characterId, u)
		if err != nil {
			return err
		}
		if !changed {
			p.l.Debugf("No stat value change for character [%d] buff [%d] stat [%s].", characterId, u.SourceId, u.StatType)
			return nil
		}
		if created {
			return buf.Put(character2.EnvEventStatusTopic, appliedStatusEventProvider(worldId, characterId, characterId, updated.SourceId(), updated.Level(), updated.Duration(), updated.Changes(), updated.CreatedAt(), updated.ExpiresAt(), updated.NoExpiry()))
		}
		return buf.Put(character2.EnvEventStatusTopic, statUpdatedStatusEventProvider(worldId, characterId, updated.SourceId(), updated.Level(), updated.Duration(), updated.Changes(), updated.CreatedAt(), updated.ExpiresAt()))
	})
}

func (p *ProcessorImpl) ExpireBuffs() error {
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, c := range GetRegistry().GetCharacters(p.ctx) {
			if err := p.expireInto(buf, c.WorldId(), c.Id()); err != nil {
				return err
			}
		}
		return nil
	})
}

// ExpireForCharacter sweeps ONE character, so a single client's CANCEL_DEBUFF
// nudge does not force a fleet-wide pass. WorldId comes from the command
// envelope — the channel knows the live session's world, which is authoritative
// for an in-session character. Semantics are identical to the fleet sweep by
// construction: both call expireInto. (task-190 FR-2.6.1)
func (p *ProcessorImpl) ExpireForCharacter(worldId world.Id, characterId uint32) error {
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		return p.expireInto(buf, worldId, characterId)
	})
}

// expireInto prunes one character's lapsed buffs and puts one EXPIRED event per
// lapsed buff on buf. Registry.GetExpired already does prune-and-return, so no
// new expiry semantics are invented here. When nothing has lapsed it puts
// nothing, and message.Emit then emits nothing — FR-2.9 / NFR-2.1 hold
// structurally, not by an explicit guard.
func (p *ProcessorImpl) expireInto(buf *message.Buffer, worldId world.Id, characterId uint32) error {
	ebs := GetRegistry().GetExpired(p.ctx, characterId)
	for _, eb := range ebs {
		p.l.Debugf("Expired buff for character [%d] from [%d].", characterId, eb.SourceId())
		if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, eb.SourceId(), eb.Level(), eb.Duration(), eb.Changes(), eb.CreatedAt(), eb.ExpiresAt(), eb.NoExpiry())); err != nil {
			return err
		}
	}
	if len(ebs) > 0 {
		sets := make([][]stat.Model, 0, len(ebs))
		for _, eb := range ebs {
			sets = append(sets, eb.Changes())
		}
		markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, sets...)
	}
	return nil
}

func ExpireBuffs(l logrus.FieldLogger, ctx context.Context) error {
	ts, err := GetRegistry().GetTenants(ctx)
	if err != nil {
		return err
	}

	for _, t := range ts {
		routine.Go(l, ctx, func(_ context.Context) {
			tctx := tenant.WithContext(ctx, t)
			if err := NewProcessor(l, tctx).ExpireBuffs(); err != nil {
				l.WithError(err).Error("Failed to expire buffs for tenant.")
			}
		})
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
		routine.Go(l, ctx, func(_ context.Context) {
			tctx := tenant.WithContext(ctx, t)
			if err := NewProcessor(l, tctx).ProcessPoisonTicks(); err != nil {
				l.WithError(err).Error("Failed to process poison ticks for tenant.")
			}
		})
	}
	return nil
}
