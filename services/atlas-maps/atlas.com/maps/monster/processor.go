package monster

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	CountInMap(transactionId uuid.UUID, field field.Model) (int, error)
	CreateMonster(transactionId uuid.UUID, field field.Model, monsterId uint32, x int16, y int16, fh int16, team int8)
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
	data, err := requests.DrainProvider[RestModel, RestModel](p.l, p.ctx)(inMapUrl(field), 250, Extract, model.Filters[RestModel]())()
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func (p *ProcessorImpl) CreateMonster(_ uuid.UUID, field field.Model, monsterId uint32, x int16, y int16, fh int16, team int8) {
	_, err := requestCreate(field, monsterId, x, y, fh, team)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Creating monster for field [%s].", field.Id())
	}
}
