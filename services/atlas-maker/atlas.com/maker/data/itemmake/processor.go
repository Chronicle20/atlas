package itemmake

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	// GetAll returns every recipe in the atlas-data catalog, draining every
	// page (see allItemMakesUrl).
	GetAll() ([]Model, error)
	// GetById returns one recipe by its produced item id, or
	// requests.ErrNotFound if atlas-data has no such recipe.
	GetById(itemId item.Id) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetAll() ([]Model, error) {
	url, err := allItemMakesUrl(p.ctx)
	if err != nil {
		return nil, err
	}
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model]())()
}

func (p *ProcessorImpl) GetById(itemId item.Id) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(p.ctx, itemId), Extract)()
}
