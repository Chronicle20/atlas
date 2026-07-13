package guild

import (
	"context"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetByMemberId(memberId uint32) (Model, error)
	ByMemberIdProvider(memberId uint32) model.Provider[[]Model]
	IsGuildMaster(characterId uint32) (bool, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetByMemberId(memberId uint32) (Model, error) {
	return model.First[Model](p.ByMemberIdProvider(memberId), model.Filters[Model]())
}

func (p *ProcessorImpl) ByMemberIdProvider(memberId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByMemberId(memberId), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) IsGuildMaster(characterId uint32) (bool, error) {
	g, err := p.GetByMemberId(characterId)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) || errors.Is(err, model.ErrEmptySlice) {
			return false, nil
		}
		return false, err
	}
	return g.IsLeader(characterId), nil
}
