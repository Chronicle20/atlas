package summon

import (
	summon2 "atlas-channel/kafka/message/summon"
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	InMapModelProvider(f field.Model) model.Provider[[]Model]
	ForEachInMap(f field.Model, o model.Operator[Model]) error
	Spawn(f field.Model, ownerCharacterId uint32, skillId uint32, level byte, x int16, y int16, auraLevel byte, hexLevel byte) error
	Move(f field.Model, summonId uint32, senderCharacterId uint32, x int16, y int16, stance byte, rawMovement []byte) error
	Attack(f field.Model, summonId uint32, senderCharacterId uint32, direction byte, targets []summon2.AttackTargetEntry) error
	Damage(f field.Model, summonId uint32, senderCharacterId uint32, damage int32, monsterIdFrom uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

// InMapModelProvider fetches the summons currently present in field f from
// atlas-summons (used to replay existing summons to a character entering
// the map). The upstream atlas-summons list is now paginated (task-117), so
// this drains every page rather than fetching just the first -- a truncated
// list here means some existing summons silently fail to replay to the
// entering character.
func (p *ProcessorImpl) InMapModelProvider(f field.Model) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(inMapUrl(f), 250, Extract, model.Filters[Model]())
}

// ForEachInMap applies o to every summon currently in field f.
func (p *ProcessorImpl) ForEachInMap(f field.Model, o model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(f), o, model.ParallelExecute())
}

// Spawn emits a COMMAND_TOPIC_SUMMON SPAWN command requesting atlas-summons
// create an owner-bound summon for the given skill at the caster's position.
// auraLevel/hexLevel carry the caster's trained AURA_OF_THE_BEHOLDER (1320008)
// and HEX_OF_THE_BEHOLDER (1320009) levels for a Beholder summon (0 otherwise);
// atlas-summons snapshots the heal/buff from them at spawn.
func (p *ProcessorImpl) Spawn(f field.Model, ownerCharacterId uint32, skillId uint32, level byte, x int16, y int16, auraLevel byte, hexLevel byte) error {
	p.l.Debugf("Requesting summon spawn for character [%d] skill [%d] level [%d] at [%d,%d].", ownerCharacterId, skillId, level, x, y)
	return producer.ProviderImpl(p.l)(p.ctx)(summon2.EnvCommandTopic)(SpawnCommandProvider(f, ownerCharacterId, skillId, level, x, y, auraLevel, hexLevel))
}

// Move emits a COMMAND_TOPIC_SUMMON MOVE command requesting atlas-summons
// reposition the given summon (ownership is verified there) and rebroadcast the
// raw movement blob byte-faithfully.
func (p *ProcessorImpl) Move(f field.Model, summonId uint32, senderCharacterId uint32, x int16, y int16, stance byte, rawMovement []byte) error {
	p.l.Debugf("Requesting summon move for summon [%d] by character [%d] to [%d,%d].", summonId, senderCharacterId, x, y)
	return producer.ProviderImpl(p.l)(p.ctx)(summon2.EnvCommandTopic)(MoveCommandProvider(f, summonId, senderCharacterId, x, y, stance, rawMovement))
}

// Attack emits a COMMAND_TOPIC_SUMMON ATTACK command requesting atlas-summons
// credit the owner, clamp the reported per-target damage, and emit an ATTACKED
// event for rebroadcast.
func (p *ProcessorImpl) Attack(f field.Model, summonId uint32, senderCharacterId uint32, direction byte, targets []summon2.AttackTargetEntry) error {
	p.l.Debugf("Requesting summon attack for summon [%d] by character [%d] against [%d] targets.", summonId, senderCharacterId, len(targets))
	return producer.ProviderImpl(p.l)(p.ctx)(summon2.EnvCommandTopic)(AttackCommandProvider(f, summonId, senderCharacterId, direction, targets))
}

// Damage emits a COMMAND_TOPIC_SUMMON DAMAGE command requesting atlas-summons
// decrement the puppet summon's HP by the reported amount (destroying it at
// zero) and emit a DAMAGED event for rebroadcast.
func (p *ProcessorImpl) Damage(f field.Model, summonId uint32, senderCharacterId uint32, damage int32, monsterIdFrom uint32) error {
	p.l.Debugf("Requesting summon damage for summon [%d] by character [%d] amount [%d] from monster [%d].", summonId, senderCharacterId, damage, monsterIdFrom)
	return producer.ProviderImpl(p.l)(p.ctx)(summon2.EnvCommandTopic)(DamageCommandProvider(f, summonId, senderCharacterId, damage, monsterIdFrom))
}
