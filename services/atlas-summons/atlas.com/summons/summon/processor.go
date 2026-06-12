package summon

import (
	"atlas-summons/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
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

type ProcessorImpl struct {
	l    logrus.FieldLogger
	ctx  context.Context
	t    tenant.Model
	emit emitter
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l: l, ctx: ctx, t: tenant.MustFromContext(ctx),
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			return producer.ProviderImpl(l)(ctx)(topic)(provider)
		},
	}
}

func (p *ProcessorImpl) GetById(id uint32) (Model, error) {
	return GetRegistry().Get(p.ctx, p.t, id)
}
func (p *ProcessorImpl) GetInField(f field.Model) ([]Model, error) {
	return GetRegistry().GetInField(p.ctx, p.t, f)
}

// Spawn is implemented in Phase 1 (roster + spawn/despawn lifecycle).
func (p *ProcessorImpl) Spawn(f field.Model, ownerCharacterId uint32, skillId uint32, skillLevel byte, x int16, y int16) (Model, error) {
	return Model{}, nil
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

// Despawn is implemented in Phase 1 (spawn/despawn lifecycle).
func (p *ProcessorImpl) Despawn(id uint32, animated bool) error {
	return nil
}

// DespawnAllForOwner is implemented in Phase 1 (logout/channel-change/map-change cascade).
func (p *ProcessorImpl) DespawnAllForOwner(ownerCharacterId uint32) error {
	return nil
}
