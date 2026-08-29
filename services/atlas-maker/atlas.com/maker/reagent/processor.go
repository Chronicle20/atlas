package reagent

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ErrNotFound is returned by GetByItemId when the tenant has no row for the
// requested item id. Callers applying reagents to a craft distinguish it from a
// genuine read failure: an unknown reagent is dropped, a failed read is not.
var ErrNotFound = errors.New("reagent not found")

type Processor interface {
	// GetAll returns every reagent the tenant owns, unpaged, for callers that
	// apply a whole set of reagents to one craft.
	GetAll() ([]Model, error)
	// GetAllPaged backs the paginated collection route.
	GetAllPaged(page model.Page) model.Provider[model.Paged[Model]]
	// GetByItemId returns the tenant's reagent for itemId, or ErrNotFound.
	GetByItemId(itemId item.Id) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, db: db}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetAll() ([]Model, error) {
	es, err := getAllProvider()(p.db.WithContext(p.ctx))()
	if err != nil {
		return nil, err
	}
	return model.SliceMap(modelFromEntity)(model.FixedProvider(es))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetAllPaged(page model.Page) model.Provider[model.Paged[Model]] {
	ep := getAllPagedProvider(page)(p.db.WithContext(p.ctx))
	return model.MapPaged(modelFromEntity)(ep)(model.ParallelMap())
}

func (p *ProcessorImpl) GetByItemId(itemId item.Id) (Model, error) {
	e, err := getByItemId(itemId)(p.db.WithContext(p.ctx))()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Model{}, fmt.Errorf("%w: item id [%d]", ErrNotFound, itemId)
		}
		return Model{}, err
	}
	return modelFromEntity(e)
}
