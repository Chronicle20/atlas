package summon

import (
	skilldata "atlas-summons/data/skill"
	"atlas-summons/data/skill/effect"
	"atlas-summons/kafka/producer"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	summonconst "github.com/Chronicle20/atlas/libs/atlas-constants/summon"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(id uint32) (Model, error)
	GetInField(f field.Model) ([]Model, error)
	Spawn(f field.Model, ownerCharacterId uint32, skillId uint32, skillLevel byte, x int16, y int16) (Model, error)
	Move(id uint32, senderCharacterId uint32, x int16, y int16, stance byte) error
	Attack(id uint32, senderCharacterId uint32, direction byte, targets []AttackTarget) error
	Damage(id uint32, senderCharacterId uint32, amount int32, monsterIdFrom uint32) error
	Despawn(id uint32, animated bool) error
	DespawnAllForOwner(ownerCharacterId uint32) error
}

// AttackTarget is one {monster, reported damage} pair from a summon-attack packet.
type AttackTarget struct {
	MonsterId uint32
	Damage    uint32
}

// emitter publishes a kafka message provider to a topic. ProcessorImpl uses this
// indirection so later phases can intercept event emissions in tests without
// spinning up kafka. Production wiring uses producer.ProviderImpl.
type emitter func(topic string, provider model.Provider[[]kafka.Message]) error

// effectSource provides per-skill effect data. The default implementation is
// the data/skill REST processor; tests substitute a stub so spawn logic is
// unit-testable without a live atlas-data.
type effectSource interface {
	GetEffect(skillId uint32, level byte) (effect.Model, error)
}

type ProcessorImpl struct {
	l       logrus.FieldLogger
	ctx     context.Context
	t       tenant.Model
	emit    emitter
	effects effectSource
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l: l, ctx: ctx, t: tenant.MustFromContext(ctx),
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			return producer.ProviderImpl(l)(ctx)(topic)(provider)
		},
		effects: skilldata.NewProcessor(l, ctx),
	}
}

func (p *ProcessorImpl) GetById(id uint32) (Model, error) {
	return GetRegistry().Get(p.ctx, p.t, id)
}
func (p *ProcessorImpl) GetInField(f field.Model) ([]Model, error) {
	return GetRegistry().GetInField(p.ctx, p.t, f)
}

// Spawn classifies the cast skill against the summon roster, removes any
// same-skill or mobility-conflicting existing summon for the owner, fetches the
// skill effect for HP/duration, persists the new summon, and emits CREATED.
// A non-summon skill is a graceful no-op (FR-1.3).
func (p *ProcessorImpl) Spawn(f field.Model, ownerCharacterId uint32, skillId uint32, skillLevel byte, x int16, y int16) (Model, error) {
	entry, ok := summonconst.Lookup(skillId)
	if !ok {
		p.l.Debugf("Skill [%d] is not a summon; no spawn.", skillId) // FR-1.3 graceful no-op
		return Model{}, nil
	}

	// FR-2.4 / FR-2.5: remove same-skill instance and conflicting-mobility-class instance.
	existing, _ := GetRegistry().GetByOwner(p.ctx, p.t, ownerCharacterId)
	for _, e := range existing {
		if e.SkillId() == skillId || conflictsMobility(entry.Movement, e.MovementType()) {
			_ = p.Despawn(e.Id(), false)
		}
	}

	eff, err := p.effects.GetEffect(skillId, skillLevel)
	if err != nil {
		p.l.WithError(err).Warnf("No effect data for summon skill [%d]; aborting spawn.", skillId)
		return Model{}, err
	}

	id := GetIdAllocator().Allocate(p.ctx, p.t)
	now := time.Now()
	expires := now.Add(time.Duration(eff.Duration()) * time.Millisecond)

	hp := int32(0)
	if entry.Type == summonconst.TypePuppet {
		hp = int32(eff.X())
	} else if entry.Type == summonconst.TypeBuffAura {
		hp = int32(eff.X()) + 1 // Cosmic Beholder hp = x + 1
	}

	b := NewBuilder().
		SetId(id).SetOwnerCharacterId(ownerCharacterId).SetSkillId(skillId).SetSkillLevel(skillLevel).
		SetSummonType(SummonType(entry.Type)).SetMovementType(MovementType(entry.Movement)).
		SetField(f).SetX(x).SetY(y).SetHp(hp).SetMaxHp(hp).
		SetSpawnTime(now).SetExpiresAt(expires).SetAnimated(true)

	// Phase 5: Beholder aura snapshot (next-heal/next-buff timers, buff changes)
	// is layered onto this builder.
	m := b.Build()

	if err := GetRegistry().Put(p.ctx, p.t, m); err != nil {
		GetIdAllocator().Release(p.ctx, p.t, id)
		return Model{}, err
	}
	if err := p.emit(EnvEventTopicSummonStatus, createdEventProvider(m)); err != nil {
		p.l.WithError(err).Errorf("Unable to emit CREATED for summon [%d].", id)
	}
	// Phase 4: emit ADD_PUPPET to atlas-monsters for puppet summons.
	// Phase 5: Beholder timer init.
	return m, nil
}

// Move is implemented in Phase 2 (movement relay).
func (p *ProcessorImpl) Move(id uint32, senderCharacterId uint32, x int16, y int16, stance byte) error {
	return nil
}

// Attack is implemented in Phase 3 (summon attack → monster damage).
func (p *ProcessorImpl) Attack(id uint32, senderCharacterId uint32, direction byte, targets []AttackTarget) error {
	return nil
}

// Damage is implemented in Phase 4 (summon takes damage).
func (p *ProcessorImpl) Damage(id uint32, senderCharacterId uint32, amount int32, monsterIdFrom uint32) error {
	return nil
}

// Despawn removes a summon from the registry, releases its oid, and emits
// DESTROYED. A missing summon is treated as already gone (no error).
func (p *ProcessorImpl) Despawn(id uint32, animated bool) error {
	m, err := GetRegistry().Get(p.ctx, p.t, id)
	if err != nil {
		return nil // already gone
	}
	if err := GetRegistry().Remove(p.ctx, p.t, id); err != nil {
		return err
	}
	GetIdAllocator().Release(p.ctx, p.t, id)
	if err := p.emit(EnvEventTopicSummonStatus, destroyedEventProvider(m, animated)); err != nil {
		p.l.WithError(err).Errorf("Unable to emit DESTROYED for summon [%d].", id)
	}
	// Phase 4: emit REMOVE_PUPPET to atlas-monsters for puppet summons.
	// Phase 5: Beholder timer cleanup is implicit (registry removal).
	return nil
}

// DespawnAllForOwner removes every summon owned by the character. Used by the
// logout / channel-change / map-change cascade.
func (p *ProcessorImpl) DespawnAllForOwner(ownerCharacterId uint32) error {
	ms, err := GetRegistry().GetByOwner(p.ctx, p.t, ownerCharacterId)
	if err != nil {
		return err
	}
	for _, m := range ms {
		_ = p.Despawn(m.Id(), false)
	}
	return nil
}

// conflictsMobility implements Cosmic StatEffect.java:1024-1029: a new stationary
// summon cancels the existing stationary one; a new non-stationary cancels the
// existing non-stationary one.
func conflictsMobility(newMove summonconst.Movement, existing MovementType) bool {
	newStationary := newMove == summonconst.MovementStationary
	existingStationary := existing == MovementStationary
	return newStationary == existingStationary
}
