package dragon

import (
	"atlas-dragons/character"
	"context"
	"errors"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	GetByCharacterId(characterId uint32) (Model, error)
	GetInField(f field.Model) ([]Model, error)
	Create(f field.Model, characterId uint32) error
	Destroy(characterId uint32) error
	Move(characterId uint32, startX int16, startY int16, stance byte, rawMovement []byte) error
}

// emitter publishes a kafka message provider to a topic. ProcessorImpl uses this
// indirection so tests can observe event emissions without a live Kafka, exactly
// as atlas-summons' summon.ProcessorImpl does.
type emitter func(t topic.Token, provider model.Provider[[]kafka.Message]) error

// characterSource resolves the owner's job and position. The default is the
// character REST processor; tests substitute a stub.
type characterSource interface {
	GetById(characterId uint32) (character.Model, error)
}

type ProcessorImpl struct {
	l          logrus.FieldLogger
	ctx        context.Context
	t          tenant.Model
	emit       emitter
	characters characterSource
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l: l, ctx: ctx, t: tenant.MustFromContext(ctx),
		emit: func(t topic.Token, provider model.Provider[[]kafka.Message]) error {
			return producer.ProviderImpl(l)(ctx)(t)(provider)
		},
		characters: character.NewProcessor(l, ctx),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Model, error) {
	return GetRegistry().Get(p.ctx, p.t, characterId)
}

func (p *ProcessorImpl) GetInField(f field.Model) ([]Model, error) {
	return GetRegistry().GetInField(p.ctx, p.t, f)
}

// Create spawns the dragon for characterId in field f, if that character's job
// is dragon-bearing. This is the ONE place the job gate is enforced, so no
// caller has to remember it.
//
// CREATED is emitted only on the absent->present transition. Kafka is
// at-least-once: a redelivered CREATE would otherwise produce a second map-wide
// SPAWN_DRAGON. The client's own release-then-recreate in
// CUser::OnDragonPacket's spawn arm is a second line of defence, not the first.
func (p *ProcessorImpl) Create(f field.Model, characterId uint32) error {
	c, err := p.characters.GetById(characterId)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			// The character is gone. Normal for a race against logout; not a
			// fetch failure and not retryable.
			p.l.Warnf("Character [%d] not found while creating dragon; skipping.", characterId)
			return nil
		}
		return err
	}

	if !HasDragon(p.t, c.JobId()) {
		return nil
	}

	existed, err := GetRegistry().Exists(p.ctx, p.t, characterId)
	if err != nil {
		return err
	}

	m := NewBuilder(characterId).
		SetField(f).
		SetX(int32(c.X())).
		SetY(int32(c.Y())).
		SetStance(c.Stance()).
		SetJobId(c.JobId()).
		Build()
	if err := GetRegistry().Put(p.ctx, p.t, m); err != nil {
		return err
	}

	if existed {
		p.l.Debugf("Dragon for character [%d] already present in map [%d]; state refreshed, no CREATED emitted.", characterId, f.MapId())
		return nil
	}
	p.l.Debugf("Created dragon for character [%d] in map [%d] at [%d,%d].", characterId, f.MapId(), m.X(), m.Y())
	return p.emit(EnvEventTopicDragonStatus, createdEventProvider(m))
}

// Destroy removes the dragon and emits DESTROYED, only if one existed.
// Destroying an absent dragon is a silent no-op returning nil (FR-1.6).
//
// Note what DESTROYED does and does not accomplish: the client has no handler
// arm for REMOVE_DRAGON, so the packet is discarded. The dragon disappears from
// other players' screens because the owner's CUser is destroyed when they leave
// the field. See the doc comment on clientbound.DragonRemove.
func (p *ProcessorImpl) Destroy(characterId uint32) error {
	m, err := GetRegistry().Get(p.ctx, p.t, characterId)
	if err != nil {
		if errors.Is(err, atlasredis.ErrNotFound) {
			return nil // no dragon; nothing to do
		}
		// A real Redis error, not "no dragon": propagating it lets the caller
		// retry instead of silently believing the (unperformed) cleanup
		// succeeded, leaving the dragon and its field-index entry orphaned.
		return err
	}
	existed, err := GetRegistry().Remove(p.ctx, p.t, characterId)
	if err != nil {
		return err
	}
	if !existed {
		return nil
	}
	p.l.Debugf("Destroyed dragon for character [%d] in map [%d].", characterId, m.Field().MapId())
	return p.emit(EnvEventTopicDragonStatus, destroyedEventProvider(m))
}

// Move updates the dragon's position and relays the raw movement blob. It
// never creates a dragon as a side effect: a move from a character with no
// dragon is dropped with a warning and no event (FR-4.4).
//
// stance is intentionally NOT persisted here. DragonMoveHandleFunc
// (atlas-channel) passes a hardcoded 0 for it: the CMovePath blob the client
// sends is opaque, and no stance is decoded from it (parsing one would need a
// full move-path codec, well beyond this scope). If that 0 were written
// through to the registry, the dragon's stored stance would go to 0 after its
// first move and stay there forever, so spawnDragonForSession would replay
// stance 0 to every late-entering player. Leaving the last known spawn
// stance in place is strictly better than zeroing it, so Move only updates
// position; the stance parameter is accepted (it is part of the Kafka
// contract, MoveCommandBody.Stance) but deliberately unused for persistence.
//
// Since the serverbound packet carries no identity field, "does this sender own
// the named dragon" is unrepresentable — the only check left is "does this
// sender have a dragon at all", which is this lookup.
func (p *ProcessorImpl) Move(characterId uint32, startX int16, startY int16, stance byte, rawMovement []byte) error {
	if _, err := GetRegistry().Get(p.ctx, p.t, characterId); err != nil {
		if !errors.Is(err, atlasredis.ErrNotFound) {
			// A real Redis error, not "no dragon": say so, don't let it read
			// as the ordinary dragonless-character case.
			p.l.WithError(err).Errorf("Move for character [%d]: lookup failed.", characterId)
			return err
		}
		p.l.Warnf("Move for character [%d] with no dragon; dropped.", characterId)
		return nil
	}
	m, err := GetRegistry().Update(p.ctx, p.t, characterId, func(cur Model) Model {
		return cur.Move(int32(startX), int32(startY))
	})
	if err != nil {
		return err
	}
	return p.emit(EnvEventTopicDragonStatus, movedEventProvider(m, rawMovement))
}
