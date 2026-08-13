package cash

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetById(itemId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(itemId uint32) (Model, error) {
	m, err := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(itemId), Extract)()
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			p.l.WithError(err).Warnf("Extender template [%d] not found in cash data; refusing to extend expiration.", itemId)
		} else {
			p.l.WithError(err).Errorf("Unable to retrieve cash data for extender template [%d].", itemId)
		}
		return Model{}, err
	}
	return m, nil
}
