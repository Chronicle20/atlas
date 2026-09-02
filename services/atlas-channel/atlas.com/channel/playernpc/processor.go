package playernpc

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor interface defines the operations for player NPC read access.
type Processor interface {
	InMapModelProvider(f field.Model) model.Provider[[]Model]
	ForEachInMap(f field.Model, o model.Operator[Model]) error
	GetInMapByObjectId(f field.Model, objectId uint32) (Model, error)
}

// ErrNotFound is returned by GetInMapByObjectId when no Player NPC with
// that object id is currently deployed in the field.
var ErrNotFound = errors.New("player npc not found")

// ProcessorImpl implements the Processor interface.
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

// InMapModelProvider fetches every Player NPC currently deployed in one
// map (used to replay existing state to a character entering the map).
// The upstream atlas-player-npcs list is paginated, so this drains every
// page rather than fetching just the first.
func (p *ProcessorImpl) InMapModelProvider(f field.Model) model.Provider[[]Model] {
	url, err := inMapUrl(p.ctx, f)
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) ForEachInMap(f field.Model, o model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(f), o, model.ParallelExecute())
}

// GetInMapByObjectId resolves one deployed Player NPC in f by its client
// object id. atlas-player-npcs exposes no by-object-id endpoint, so this
// filters the map list -- the same read the map-enter replay already makes,
// over a set bounded by the map's script-id band capacity. Returns
// ErrNotFound when the object is not deployed in f.
func (p *ProcessorImpl) GetInMapByObjectId(f field.Model, objectId uint32) (Model, error) {
	ns, err := p.InMapModelProvider(f)()
	if err != nil {
		return Model{}, err
	}
	for _, n := range ns {
		if n.ObjectId() == objectId {
			return n, nil
		}
	}
	return Model{}, ErrNotFound
}
