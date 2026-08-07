package item

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor is the channel-side read client for atlas-data's item-string search
// index. It backs the marketplace SEARCH_ITC_LIST arm: a search term is resolved
// to the set of matching item template ids, which the browse then filters on.
type Processor interface {
	ByNameProvider(query string) model.Provider[[]Model]
	GetIdsByName(query string) ([]uint32, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) ByNameProvider(query string) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByName(query), Extract, model.Filters[Model]())
}

// GetIdsByName resolves a search term to the matching item template ids via the
// atlas-data item-string search index. A REST/decode error is returned to the
// caller (the search arm falls back to an empty result on error). The returned
// slice may be empty when nothing matches.
func (p *ProcessorImpl) GetIdsByName(query string) ([]uint32, error) {
	ms, err := p.ByNameProvider(query)()
	if err != nil {
		return nil, err
	}
	ids := make([]uint32, 0, len(ms))
	for _, m := range ms {
		ids = append(ids, m.ItemId())
	}
	return ids, nil
}
