package character

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	// AllProvider returns a provider for all characters of the tenant in context.
	AllProvider() model.Provider[[]Model]
	// GetAll returns all characters of the tenant in context.
	GetAll() ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// AllProvider fetches every character of the tenant in context.
// atlas-character's GET /characters is paginated (default page size 50);
// ranking computation is a genuine semantic-"all" consumer, so this drains
// every page via requests.DrainProvider rather than reading only the first
// page — see tenant.ProcessorImpl.AllProvider for the identical rationale
// and page-size choice.
func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(allCharactersUrl(), 250, Extract, model.Filters[Model]())
}

// GetAll returns all characters of the tenant in context.
func (p *ProcessorImpl) GetAll() ([]Model, error) {
	return p.AllProvider()()
}
