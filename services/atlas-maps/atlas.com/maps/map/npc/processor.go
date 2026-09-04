package npc

import (
	npcKafka "atlas-maps/kafka/message/npc"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	Create(f field.Model, input RestModel) (Model, error)
	GetInField(f field.Model) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, t: tenant.MustFromContext(ctx)}
}

var _ Processor = (*ProcessorImpl)(nil)

// Create places a scripted NPC on a field. When input.SpawnIfAbsent is set
// and an NPC with the same NpcId is already present on this field, the call
// is a no-op: it returns a zero Model (UniqueId() == 0) and a nil error — the
// same idempotency shape Plan A Task A7 landed for atlas-monsters' Create
// (services/atlas-monsters/atlas.com/monsters/monster/processor.go).
func (p *ProcessorImpl) Create(f field.Model, input RestModel) (Model, error) {
	if input.SpawnIfAbsent {
		existing, err := p.GetInField(f)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to check field [%s] for an existing npc [%d].", f.Id(), input.NpcId)
			return Model{}, err
		}
		for _, n := range existing {
			if n.NpcId() == input.NpcId {
				p.l.Debugf("Suppressing spawn of npc [%d] in field [%s]: already present as [%d].", input.NpcId, f.Id(), n.UniqueId())
				return Model{}, nil
			}
		}
	}

	key := FieldKey{Tenant: p.t, Field: f}
	m := NewModel(f, getRegistry().NextId(), input.NpcId, input.X, input.Y, input.Fh)
	getRegistry().Add(key, m)

	p.l.Debugf("Created npc [%d] with id [%d] in field [%s]. Emitting NPC Status.", input.NpcId, m.UniqueId(), f.Id())
	if err := emit(p.l, p.ctx, npcKafka.EnvEventTopicNpcStatus, CreatedEventProvider(f, m)); err != nil {
		p.l.WithError(err).Errorf("Unable to emit npc created event for npc [%d] in field [%s].", m.UniqueId(), f.Id())
	}
	return m, nil
}

// GetInField returns the scripted NPCs currently placed on the given field.
func (p *ProcessorImpl) GetInField(f field.Model) ([]Model, error) {
	key := FieldKey{Tenant: p.t, Field: f}
	return getRegistry().GetAll(key), nil
}
