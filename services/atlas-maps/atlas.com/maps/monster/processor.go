package monster

import (
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	CountInMap(transactionId uuid.UUID, field field.Model) (int, error)
	CreateMonster(transactionId uuid.UUID, field field.Model, monsterId uint32, x int16, y int16, fh uint16, team int32)
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

func (p *ProcessorImpl) CountInMap(transactionId uuid.UUID, field field.Model) (int, error) {
	data, err := requestInMap(field)(p.l, p.ctx)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func (p *ProcessorImpl) CreateMonster(transactionId uuid.UUID, field field.Model, monsterId uint32, x int16, y int16, fh uint16, team int32) {
	_, err := requestCreate(field, monsterId, x, y, fh, team)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Creating monster for field [%s].", field.Id())
	}
}
