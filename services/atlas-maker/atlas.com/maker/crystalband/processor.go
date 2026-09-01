package crystalband

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ErrNotFound is returned by GetByMinLevel and CrystalForLevel when the
// tenant has no matching row. For CrystalForLevel specifically: the
// derivation (docs/tasks/task-285-maker-skill-crafting/reagent-derivation.md
// §5.5) proves CItemMakerInfo's monster-crystal band vector is write-only in
// both gms_v72 and gms_v83 — nothing in the client ever reads it back — so
// out-of-band reqLevel has no client behaviour to reproduce. Atlas's own
// ruling is to reject the craft rather than clamp: reqLevel outside every
// seeded band returns ErrNotFound, the same sentinel shape reagent exposes so
// Task 23 can distinguish it from a genuine read failure.
var ErrNotFound = errors.New("crystal band not found")

type Processor interface {
	// GetAll returns every crystal band the tenant owns, unpaged.
	GetAll() ([]Model, error)
	// GetAllPaged backs the paginated collection route.
	GetAllPaged(page model.Page) model.Provider[model.Paged[Model]]
	// GetByMinLevel returns the tenant's band starting at minLevel, or
	// ErrNotFound.
	GetByMinLevel(minLevel uint32) (Model, error)
	// CrystalForLevel returns the crystal item id and count for an equip
	// with reqLevel as its level requirement, or ErrNotFound if reqLevel
	// falls outside every seeded band (see ErrNotFound's doc comment).
	CrystalForLevel(reqLevel uint32) (item.Id, uint32, error)
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
	return model.SliceMap(Make)(model.FixedProvider(es))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetAllPaged(page model.Page) model.Provider[model.Paged[Model]] {
	ep := getAllPagedProvider(page)(p.db.WithContext(p.ctx))
	return model.MapPaged(Make)(ep)(model.ParallelMap())
}

func (p *ProcessorImpl) GetByMinLevel(minLevel uint32) (Model, error) {
	e, err := getByMinLevel(minLevel)(p.db.WithContext(p.ctx))()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Model{}, fmt.Errorf("%w: min level [%d]", ErrNotFound, minLevel)
		}
		return Model{}, err
	}
	return Make(e)
}

// CrystalForLevel loads the tenant's bands once and resolves reqLevel in
// memory, rather than issuing a query per lookup.
func (p *ProcessorImpl) CrystalForLevel(reqLevel uint32) (item.Id, uint32, error) {
	ms, err := p.GetAll()
	if err != nil {
		return 0, 0, err
	}
	for _, m := range ms {
		if m.Contains(reqLevel) {
			return m.CrystalItemId(), m.Count(), nil
		}
	}
	return 0, 0, fmt.Errorf("%w: level [%d]", ErrNotFound, reqLevel)
}
