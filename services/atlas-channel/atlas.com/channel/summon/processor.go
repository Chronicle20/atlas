package summon

import (
	summon2 "atlas-channel/kafka/message/summon"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

// Spawn emits a COMMAND_TOPIC_SUMMON SPAWN command requesting atlas-summons
// create an owner-bound summon for the given skill at the caster's position.
func (p *Processor) Spawn(f field.Model, ownerCharacterId uint32, skillId uint32, level byte, x int16, y int16) error {
	p.l.Debugf("Requesting summon spawn for character [%d] skill [%d] level [%d] at [%d,%d].", ownerCharacterId, skillId, level, x, y)
	return producer.ProviderImpl(p.l)(p.ctx)(summon2.EnvCommandTopic)(SpawnCommandProvider(f, ownerCharacterId, skillId, level, x, y))
}

// Move emits a COMMAND_TOPIC_SUMMON MOVE command requesting atlas-summons
// reposition the given summon (ownership is verified there) and rebroadcast the
// raw movement blob byte-faithfully.
func (p *Processor) Move(f field.Model, summonId uint32, senderCharacterId uint32, x int16, y int16, stance byte, rawMovement []byte) error {
	p.l.Debugf("Requesting summon move for summon [%d] by character [%d] to [%d,%d].", summonId, senderCharacterId, x, y)
	return producer.ProviderImpl(p.l)(p.ctx)(summon2.EnvCommandTopic)(MoveCommandProvider(f, summonId, senderCharacterId, x, y, stance, rawMovement))
}
