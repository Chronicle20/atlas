package monster

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	CountInMap(transactionId uuid.UUID, field field.Model) (int, error)
	CreateMonster(transactionId uuid.UUID, field field.Model, monsterId uint32, x int16, y int16, fh int16, team int8)
	GetInMapRect(f field.Model, x1, y1, x2, y2 int16, limit uint32, configurators ...requests.Configurator) ([]RestModel, error)
	DeleteInMap(f field.Model) error
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) CountInMap(_ uuid.UUID, field field.Model) (int, error) {
	url, err := inMapUrl(p.ctx, field)
	if err != nil {
		return 0, err
	}
	data, err := requests.DrainProvider[RestModel, RestModel](p.l, p.ctx)(url, 250, Extract, model.Filters[RestModel]())()
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func (p *ProcessorImpl) CreateMonster(_ uuid.UUID, field field.Model, monsterId uint32, x int16, y int16, fh int16, team int8) {
	_, err := requestCreate(p.ctx, field, monsterId, x, y, fh, team)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Creating monster for field [%s].", field.Id())
	}
}

// GetInMapRect returns every monster whose position falls inside the inclusive
// world-coordinate rectangle. The atlas-monsters endpoint is authoritative for
// the containment test -- callers must NOT re-filter the result, because a
// second filter with a different edge convention would silently disagree with
// the server and mask any endpoint bug. One authority per question.
//
// limit == 0 means "no cap". The result is drained across all pages.
//
// configurators are forwarded to every page's GET (e.g. requests.SetTimeout
// to bound a caller on a tight tick budget); callers on the default REST
// timeout can omit them.
func (p *ProcessorImpl) GetInMapRect(f field.Model, x1, y1, x2, y2 int16, limit uint32, configurators ...requests.Configurator) ([]RestModel, error) {
	url, err := inMapRectUrl(p.ctx, f, x1, y1, x2, y2, limit)
	if err != nil {
		return nil, err
	}
	return requests.DrainProvider[RestModel, RestModel](p.l, p.ctx)(url, 250, Extract, model.Filters[RestModel](), configurators...)()
}

// DeleteInMap removes every monster currently alive in field f, via
// atlas-monsters' `DELETE .../monsters` route -- the whole-field monster
// clear that field.ResetField composes with (Cosmic's clearMapObjects()
// monster half).
func (p *ProcessorImpl) DeleteInMap(f field.Model) error {
	url, err := inMapUrl(p.ctx, f)
	if err != nil {
		return err
	}
	return requests.DeleteRequest(url)(p.l, p.ctx)
}
