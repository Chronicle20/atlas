package chair

import (
	_map2 "atlas-chairs/data/map"
	setup2 "atlas-chairs/data/setup"
	"atlas-chairs/kafka/message"
	chair2 "atlas-chairs/kafka/message/chair"
	character2 "atlas-chairs/kafka/message/character"
	"atlas-chairs/validation"
	"context"
	"errors"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"math"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

// minRecoveryTickIntervalMillis is the server-side floor between honored
// recovery ticks per stat per character. The client cadence is frame-paced
// (accumulator +30/frame, threshold 10000): ~11.1 s at 30 fps, ~5.6 s at
// 60 fps, ~4.5 s at 75 fps (task-141 design §7). 4000 ms sits comfortably
// below the fastest legitimate cadence while capping spam at ~15 ticks/min.
// Server-internal policy, not a client-wire value (DOM-25 does not apply).
const minRecoveryTickIntervalMillis int64 = 4000

type Processor interface {
	GetById(characterId uint32) (Model, error)
	Set(field field.Model, chairType string, chairId uint32, characterId uint32) error
	Clear(field field.Model, characterId uint32) error
	RecoverAndEmit(field field.Model, characterId uint32, claimedHp int16, claimedMp int16) error
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
	m, ok := GetRegistry().Get(p.ctx, characterId)
	if !ok {
		return Model{}, errors.New("not found")
	}
	return m, nil
}

func (p *ProcessorImpl) Set(field field.Model, chairType string, chairId uint32, characterId uint32) error {
	p.l.Debugf("Attempting to sit in chair [%d] for character [%d].", chairId, characterId)
	_, err := p.GetById(characterId)
	if err == nil {
		p.l.Errorf("Character [%d] already sitting on chair.", characterId)
		_ = producer.ProviderImpl(p.l)(p.ctx)(chair2.EnvEventTopicStatus)(statusEventErrorProvider(field, chairType, chairId, characterId, chair2.ErrorTypeAlreadySitting))
		return errors.New("already sitting")
	}

	if chairType == chair2.ChairTypeFixed {
		var m _map2.Model
		m, err = _map2.NewProcessor(p.l, p.ctx).GetById(field.MapId())
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve map [%d] character [%d] is sitting in.", field.MapId(), characterId)
			_ = producer.ProviderImpl(p.l)(p.ctx)(chair2.EnvEventTopicStatus)(statusEventErrorProvider(field, chairType, chairId, characterId, chair2.ErrorTypeInternal))
			return err
		}

		if chairId >= m.Seats() {
			p.l.Errorf("Character [%d] is attempting to sit in fixed chair [%d] in map [%d], but that does not exist.", characterId, chairId, field.MapId())
			_ = producer.ProviderImpl(p.l)(p.ctx)(chair2.EnvEventTopicStatus)(statusEventErrorProvider(field, chairType, chairId, characterId, chair2.ErrorTypeDoesNotExist))
			return errors.New("chair does not exist")
		}

	}
	if chairType == chair2.ChairTypePortable {
		itemCategory := uint32(math.Floor(float64(chairId / 10000)))
		if itemCategory != 301 {
			p.l.Errorf("Character [%d] is attempting to sit in portable chair [%d] in map [%d], but that does not exist.", characterId, chairId, field.MapId())
			_ = producer.ProviderImpl(p.l)(p.ctx)(chair2.EnvEventTopicStatus)(statusEventErrorProvider(field, chairType, chairId, characterId, chair2.ErrorTypeDoesNotExist))
			return errors.New("chair does not exist")
		}

		hasItem, err := validation.NewProcessor(p.l, p.ctx).HasItem(characterId, chairId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to validate item ownership for character [%d].", characterId)
			_ = producer.ProviderImpl(p.l)(p.ctx)(chair2.EnvEventTopicStatus)(statusEventErrorProvider(field, chairType, chairId, characterId, chair2.ErrorTypeInternal))
			return err
		}
		if !hasItem {
			p.l.Errorf("Character [%d] does not own portable chair [%d].", characterId, chairId)
			_ = producer.ProviderImpl(p.l)(p.ctx)(chair2.EnvEventTopicStatus)(statusEventErrorProvider(field, chairType, chairId, characterId, chair2.ErrorTypeNotOwned))
			return errors.New("character does not own chair")
		}
		p.l.Debugf("Character [%d] validated ownership of portable chair [%d].", characterId, chairId)
	}

	c := Model{
		id:        chairId,
		chairType: chairType,
	}

	GetRegistry().Set(p.ctx, characterId, c)
	return producer.ProviderImpl(p.l)(p.ctx)(chair2.EnvEventTopicStatus)(statusEventUsedProvider(field, chairType, chairId, characterId))
}

func (p *ProcessorImpl) Clear(field field.Model, characterId uint32) error {
	p.l.Debugf("Attempting to clear for character [%d].", characterId)
	c, err := p.GetById(characterId)
	if err != nil {
		p.l.WithError(err).Debugf("Failed to get chair for character [%d].", characterId)
		return err
	}
	existed := GetRegistry().Clear(p.ctx, characterId)
	if existed {
		return producer.ProviderImpl(p.l)(p.ctx)(chair2.EnvEventTopicStatus)(statusEventCancelledProvider(field, c.Type(), c.Id(), characterId))
	}
	return nil
}

func (p *ProcessorImpl) RecoverAndEmit(f field.Model, characterId uint32, claimedHp int16, claimedMp int16) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.Recover(buf)(f, characterId, claimedHp, claimedMp)
	})
}

// Recover validates a HEAL_OVER_TIME tick (task-141 design §5.3). Seated on a
// portable chair with recovery stats: the item value is applied (the claim is
// ignored) at most once per minRecoveryTickIntervalMillis per stat. Everything
// else passes the claimed values through unchanged, preserving the pre-task-141
// natural-regen behavior (including negative jms clamp-to-max corrections).
func (p *ProcessorImpl) Recover(mb *message.Buffer) func(f field.Model, characterId uint32, claimedHp int16, claimedMp int16) error {
	return func(f field.Model, characterId uint32, claimedHp int16, claimedMp int16) error {
		m, err := p.GetById(characterId)
		if err != nil || m.Type() != chair2.ChairTypePortable {
			return p.passThrough(mb, f, characterId, claimedHp, claimedMp)
		}

		s, err := setup2.NewProcessor(p.l, p.ctx).GetById(m.Id())
		if err != nil {
			p.l.WithError(err).Warnf("Unable to retrieve setup data for chair [%d]; dropping recovery tick for character [%d].", m.Id(), characterId)
			return nil
		}
		if s.RecoveryHP() == 0 && s.RecoveryMP() == 0 {
			return p.passThrough(mb, f, characterId, claimedHp, claimedMp)
		}

		now := time.Now().UnixMilli()
		updated := m

		if s.RecoveryHP() > 0 {
			if now-updated.LastHpRecoveryAt() < minRecoveryTickIntervalMillis {
				p.l.Debugf("Dropping HP recovery tick for character [%d] on chair [%d]: reason [rate].", characterId, m.Id())
			} else {
				if claimedHp != int16(s.RecoveryHP()) {
					p.l.Warnf("Character [%d] claimed HP recovery [%d] differing from chair [%d] item value [%d]; applying item value.", characterId, claimedHp, m.Id(), s.RecoveryHP())
				}
				if err = mb.Put(character2.EnvCommandTopic, changeHPCommandProvider(f, characterId, int16(s.RecoveryHP()))); err != nil {
					return err
				}
				updated = updated.WithHpRecoveryAt(now)
			}
		} else if claimedHp != 0 {
			if err = mb.Put(character2.EnvCommandTopic, changeHPCommandProvider(f, characterId, claimedHp)); err != nil {
				return err
			}
		}

		if s.RecoveryMP() > 0 {
			if now-updated.LastMpRecoveryAt() < minRecoveryTickIntervalMillis {
				p.l.Debugf("Dropping MP recovery tick for character [%d] on chair [%d]: reason [rate].", characterId, m.Id())
			} else {
				if claimedMp != int16(s.RecoveryMP()) {
					p.l.Warnf("Character [%d] claimed MP recovery [%d] differing from chair [%d] item value [%d]; applying item value.", characterId, claimedMp, m.Id(), s.RecoveryMP())
				}
				if err = mb.Put(character2.EnvCommandTopic, changeMPCommandProvider(f, characterId, int16(s.RecoveryMP()))); err != nil {
					return err
				}
				updated = updated.WithMpRecoveryAt(now)
			}
		} else if claimedMp != 0 {
			if err = mb.Put(character2.EnvCommandTopic, changeMPCommandProvider(f, characterId, claimedMp)); err != nil {
				return err
			}
		}

		if updated != m {
			GetRegistry().Set(p.ctx, characterId, updated)
		}
		return nil
	}
}

func (p *ProcessorImpl) passThrough(mb *message.Buffer, f field.Model, characterId uint32, claimedHp int16, claimedMp int16) error {
	if claimedHp != 0 {
		if err := mb.Put(character2.EnvCommandTopic, changeHPCommandProvider(f, characterId, claimedHp)); err != nil {
			return err
		}
	}
	if claimedMp != 0 {
		if err := mb.Put(character2.EnvCommandTopic, changeMPCommandProvider(f, characterId, claimedMp)); err != nil {
			return err
		}
	}
	return nil
}
