package door

import (
	doormsg "atlas-channel/kafka/message/door"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	InFieldModelProvider(f field.Model) model.Provider[[]Model]
	GetInField(f field.Model) ([]Model, error)
	ByOwnerModelProvider(ownerCharacterId uint32) model.Provider[[]Model]
	GetByOwner(ownerCharacterId uint32) ([]Model, error)
	ForEachInMap(f field.Model, o model.Operator[Model]) error
	GetByOwnerOnMap(f field.Model, ownerCharacterId uint32) (Model, bool)
	Spawn(f field.Model, ownerCharacterId, skillId uint32, level byte, x, y int16) error
	Remove(f field.Model, ownerCharacterId uint32, reason string) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) InFieldModelProvider(f field.Model) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInField(f), Extract, model.Filters[Model]())
}

// GetInField returns all doors in the given field.
func (p *ProcessorImpl) GetInField(f field.Model) ([]Model, error) {
	return p.InFieldModelProvider(f)()
}

func (p *ProcessorImpl) ByOwnerModelProvider(ownerCharacterId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByOwner(ownerCharacterId), Extract, model.Filters[Model]())
}

// GetByOwner returns all live doors owned by ownerCharacterId, resolved from
// either side (area or town) via the atlas-doors by-owner route.
func (p *ProcessorImpl) GetByOwner(ownerCharacterId uint32) ([]Model, error) {
	return p.ByOwnerModelProvider(ownerCharacterId)()
}

// ForEachInMap applies op to every door in the given field (area-keyed).
// Mirrors reactor.Processor.ForEachInMap.
func (p *ProcessorImpl) ForEachInMap(f field.Model, o model.Operator[Model]) error {
	return model.ForEachSlice(p.InFieldModelProvider(f), o, model.ParallelExecute())
}

// GetByOwnerOnMap returns the door in the field owned by ownerCharacterId, if any.
func (p *ProcessorImpl) GetByOwnerOnMap(f field.Model, ownerCharacterId uint32) (Model, bool) {
	ms, err := p.GetInField(f)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve doors in field [%d].", f.MapId())
		return Model{}, false
	}
	for _, m := range ms {
		if m.OwnerCharacterId() == ownerCharacterId {
			return m, true
		}
	}
	return Model{}, false
}

// Spawn emits a SPAWN command to atlas-doors for a newly cast Mystic Door.
func (p *ProcessorImpl) Spawn(f field.Model, ownerCharacterId, skillId uint32, level byte, x, y int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(doormsg.EnvDoorCommandTopic)(SpawnCommandProvider(f, ownerCharacterId, skillId, level, x, y))
}

// Remove emits a REMOVE command to atlas-doors for the owner's door — used when
// the caster cancels the Mystic Door buff (the door is dismissed early).
func (p *ProcessorImpl) Remove(f field.Model, ownerCharacterId uint32, reason string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(doormsg.EnvDoorCommandTopic)(RemoveCommandProvider(f, ownerCharacterId, reason))
}
