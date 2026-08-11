package kite

import (
	"atlas-kites/character"
	"atlas-kites/configuration"
	"atlas-kites/kafka/message"
	kiteMsg "atlas-kites/kafka/message/kite"
	"context"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Typed refusal errors so consumers can log the reason without string
// matching (FR-5).
var (
	ErrMapForbidden   = errors.New("map forbidden")
	ErrMessageTooLong = errors.New("message too long")
	ErrAlreadyPlaced  = errors.New("already placed")
	ErrMapFull        = errors.New("map full")
)

// ErrNotFound is returned when an operation targets a character with no
// placed kite.
var ErrNotFound = errors.New("kite not found")

// Processor is the authoritative kite lifecycle: every FR-5 placement
// refusal is evaluated here, not in atlas-channel, so the registry enforces
// its own invariants and two concurrent requests cannot both pass a
// channel-side check.
type Processor interface {
	Create(mb *message.Buffer) func(f field.Model, characterId uint32, cmd kiteMsg.CreateCommandBody) (Model, error)
	CreateAndEmit(f field.Model, characterId uint32, cmd kiteMsg.CreateCommandBody) (Model, error)
	Destroy(mb *message.Buffer) func(characterId uint32, reason string) (Model, error)
	DestroyAndEmit(characterId uint32, reason string) (Model, error)
	GetByCharacterId(characterId uint32) (Model, error)
	InMapModelProvider(f field.Model) model.Provider[[]Model]
	GetInMap(f field.Model) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	p   producer.Provider
	r   *Registry
}

// NewProcessor constructs the canonical Processor wired to the singleton
// registry and the project's standard producer.Provider.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return NewProcessorWithProvider(l, ctx, producer.ProviderImpl(l)(ctx))
}

// NewProcessorWithProvider constructs a Processor with an injected
// producer.Provider. This is the test seam: tests supply a recording
// provider so Create/Destroy's emitted events can be asserted without a
// live Kafka broker. A nil logger (as tests pass) is replaced with a
// discard-output logger rather than left nil, since a nil FieldLogger
// interface panics on every method call, including the refusal-path logging
// in Create.
func NewProcessorWithProvider(l logrus.FieldLogger, ctx context.Context, p producer.Provider) Processor {
	if l == nil {
		dl := logrus.New()
		dl.SetOutput(io.Discard)
		l = dl
	}
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		p:   p,
		r:   getRegistry(),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// Create validates placement, allocates the wire id, inserts, and buffers
// KITE_CREATED. Every refusal buffers KITE_CREATION_FAILED instead and returns
// a typed error; none of them emits KITE_CREATED (FR-3.5).
//
// Order matters. The map-policy, message-length and one-per-character checks
// are character-local and run BEFORE the field lock, so a refusal never
// contends. Only the per-map cap needs the lock: the command topic is keyed on
// characterId, so one character's commands are totally ordered within a
// partition and FR-5.2 is safe by construction, but two DIFFERENT characters
// placing on the same full-but-for-one map land on different partitions.
func (p *ProcessorImpl) Create(mb *message.Buffer) func(f field.Model, characterId uint32, cmd kiteMsg.CreateCommandBody) (Model, error) {
	return func(f field.Model, characterId uint32, cmd kiteMsg.CreateCommandBody) (Model, error) {
		cfg := configuration.GetRegistry().GetTenantConfig(p.l, p.ctx, p.t.Id())

		refuse := func(reason string, err error) (Model, error) {
			p.l.WithFields(logrus.Fields{
				"tenant": p.t.Id().String(), "character": characterId,
				"world": f.WorldId(), "channel": f.ChannelId(),
				"map": f.MapId(), "instance": f.Instance().String(),
				"reason": reason,
			}).Infof("Refusing kite placement.")
			if bufErr := mb.Put(kiteMsg.EnvEventTopicStatus,
				creationFailedStatusEventProvider(uuid.New(), f, characterId, reason)); bufErr != nil {
				return Model{}, bufErr
			}
			return Model{}, err
		}

		if cfg.IsMapBlocked(f.MapId()) {
			return refuse(kiteMsg.FailureReasonMapForbidden, ErrMapForbidden)
		}
		if len(cmd.Message) > cfg.MaxMessageLength() {
			return refuse(kiteMsg.FailureReasonMessageTooLong, ErrMessageTooLong)
		}
		if exists, err := p.r.Exists(p.ctx, characterId); err != nil {
			return Model{}, err
		} else if exists {
			return refuse(kiteMsg.FailureReasonAlreadyPlaced, ErrAlreadyPlaced)
		}

		locked, err := p.r.AcquireFieldLock(p.ctx, f)
		if err != nil {
			return Model{}, err
		}
		if !locked {
			// Logged distinctly so contention is separable from a genuinely
			// full map; a lost race on a full map refuses either way.
			p.l.Debugf("Kite field lock contended for character [%d]; refusing as MAP_FULL.", characterId)
			return refuse(kiteMsg.FailureReasonMapFull, ErrMapFull)
		}
		defer func() {
			if relErr := p.r.ReleaseFieldLock(p.ctx, f); relErr != nil {
				p.l.WithError(relErr).Warnf("Unable to release kite field lock for map [%d].", f.MapId())
			}
		}()

		inField, err := p.GetInMap(f)
		if err != nil {
			return Model{}, err
		}
		if len(inField) >= cfg.MaxPerMap() {
			return refuse(kiteMsg.FailureReasonMapFull, ErrMapFull)
		}

		id, err := p.r.NextId(p.ctx)
		if err != nil {
			return Model{}, err
		}
		m := NewBuilder(id, f, characterId).
			SetName(cmd.Name).
			SetTemplateId(cmd.TemplateId).
			SetMessage(cmd.Message).
			SetPosition(cmd.X, cmd.Y).
			SetCreatedAt(time.Now()).
			Build()

		if err = p.r.Put(p.ctx, m); err != nil {
			return Model{}, err
		}
		if err = mb.Put(kiteMsg.EnvEventTopicStatus, createdStatusEventProvider(uuid.New(), m)); err != nil {
			return Model{}, err
		}
		return m, nil
	}
}

// isRefusal reports whether err is one of the four typed FR-5 policy
// refusals. A refusal is NOT an emit failure: its CREATION_FAILED event is
// already in the buffer and must still be flushed, so CreateAndEmit lets the
// flush complete and re-raises the refusal afterwards.
func isRefusal(err error) bool {
	return errors.Is(err, ErrMapForbidden) ||
		errors.Is(err, ErrMessageTooLong) ||
		errors.Is(err, ErrAlreadyPlaced) ||
		errors.Is(err, ErrMapFull)
}

// CreateAndEmit rolls the registry insert back when the emit fails, so the
// registry never holds a kite downstream consumers will not see
// (services/atlas-maps/atlas.com/maps/mist/processor.go:94-106).
func (p *ProcessorImpl) CreateAndEmit(f field.Model, characterId uint32, cmd kiteMsg.CreateCommandBody) (Model, error) {
	var m Model
	var refusal error

	err := message.Emit(p.p)(func(buf *message.Buffer) error {
		var innerErr error
		m, innerErr = p.Create(buf)(f, characterId, cmd)
		if isRefusal(innerErr) {
			refusal = innerErr
			return nil
		}
		return innerErr
	})
	if err != nil {
		// Only reached when the insert succeeded and the flush failed; Remove
		// on a character with no kite is a no-op, so this is safe for the
		// pre-insert error paths too.
		_ = p.r.Remove(p.ctx, characterId)
		return Model{}, err
	}
	if refusal != nil {
		return Model{}, refusal
	}
	return m, nil
}

// Destroy removes the character's kite and buffers KITE_DESTROYED, keyed on
// the field the kite was placed in. The field is read off the kite BEFORE
// the registry removal so the correct map still receives the event.
func (p *ProcessorImpl) Destroy(mb *message.Buffer) func(characterId uint32, reason string) (Model, error) {
	return func(characterId uint32, reason string) (Model, error) {
		m, ok, err := p.r.Get(p.ctx, characterId)
		if err != nil {
			return Model{}, err
		}
		if !ok {
			return Model{}, ErrNotFound
		}
		if err = p.r.Remove(p.ctx, characterId); err != nil {
			return Model{}, err
		}
		if err = mb.Put(kiteMsg.EnvEventTopicStatus, destroyedStatusEventProvider(uuid.New(), m, reason)); err != nil {
			return Model{}, err
		}
		return m, nil
	}
}

// DestroyAndEmit treats the registry removal as authoritative: once the kite
// is gone from the registry, a downstream emit failure is logged rather than
// failing the call (mist Destroy precedent,
// services/atlas-maps/atlas.com/maps/mist/processor.go:112-123). A genuine
// domain error (no such kite, a Redis failure) is distinguished from an emit
// failure and always returned.
func (p *ProcessorImpl) DestroyAndEmit(characterId uint32, reason string) (Model, error) {
	var m Model
	var domainErr error

	emitErr := message.Emit(p.p)(func(buf *message.Buffer) error {
		var innerErr error
		m, innerErr = p.Destroy(buf)(characterId, reason)
		if innerErr != nil {
			domainErr = innerErr
			return nil
		}
		return nil
	})
	if domainErr != nil {
		return Model{}, domainErr
	}
	if emitErr != nil {
		p.l.WithError(emitErr).Errorf("Unable to emit KITE_DESTROYED for character [%d]; removal already applied.", characterId)
	}
	return m, nil
}

// GetByCharacterId returns the kite owned by characterId, or ErrNotFound if
// the character has none placed.
func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Model, error) {
	m, ok, err := p.r.Get(p.ctx, characterId)
	if err != nil {
		return Model{}, err
	}
	if !ok {
		return Model{}, ErrNotFound
	}
	return m, nil
}

// InMapModelProvider composes the character index for f with kite ownership:
// it filters the characters currently in the field down to the ones that own
// a kite, then maps them through GetByCharacterId. This is exactly
// chalkboard/resource.go:71-92's InMapProvider+FilteredProvider composition,
// not a second index -- the kite registry is keyed by characterId already.
func (p *ProcessorImpl) InMapModelProvider(f field.Model) model.Provider[[]Model] {
	cip := character.NewProcessor(p.l, p.ctx).InMapProvider(f)
	fcip := model.FilteredProvider(cip, model.Filters[uint32](func(cid uint32) bool {
		exists, err := p.r.Exists(p.ctx, cid)
		return err == nil && exists
	}))
	return model.SliceMap[uint32, Model](p.GetByCharacterId)(fcip)(model.ParallelMap())
}

// GetInMap returns every kite currently placed in field f.
func (p *ProcessorImpl) GetInMap(f field.Model) ([]Model, error) {
	return p.InMapModelProvider(f)()
}
