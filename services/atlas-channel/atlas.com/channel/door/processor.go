package door

import (
	doormsg "atlas-channel/kafka/message/door"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx}
}

func (p *Processor) InFieldModelProvider(f field.Model) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInField(f), Extract, model.Filters[Model]())
}

// GetInField returns all doors in the given field.
func (p *Processor) GetInField(f field.Model) ([]Model, error) {
	return p.InFieldModelProvider(f)()
}

// GetByOwnerOnMap returns the door in the field owned by ownerCharacterId, if any.
func (p *Processor) GetByOwnerOnMap(f field.Model, ownerCharacterId uint32) (Model, bool) {
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
func (p *Processor) Spawn(f field.Model, ownerCharacterId, skillId uint32, level byte, x, y int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(doormsg.EnvDoorCommandTopic)(SpawnCommandProvider(f, ownerCharacterId, skillId, level, x, y))
}
