package ranking

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	// ByCharacterIdsProvider returns a provider for rankings of the given characters.
	ByCharacterIdsProvider(ids []uint32) model.Provider[[]Model]
	// GetByCharacterIds bulk-fetches rankings for the given characters in a
	// single call. Characters with no computed ranking are simply absent
	// from the result — callers must not treat that as an error.
	GetByCharacterIds(ids []uint32) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) ByCharacterIdsProvider(ids []uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacterIds(ids), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacterIds(ids []uint32) ([]Model, error) {
	return p.ByCharacterIdsProvider(ids)()
}
