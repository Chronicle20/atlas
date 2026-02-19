package guild

import (
	"context"
	"errors"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx}
}

func (p *Processor) GetByMemberId(memberId uint32) (Model, error) {
	return model.First[Model](p.ByMemberIdProvider(memberId), model.Filters[Model]())
}

func (p *Processor) ByMemberIdProvider(memberId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByMemberId(memberId), Extract, model.Filters[Model]())
}

func (p *Processor) IsGuildMaster(characterId uint32) (bool, error) {
	g, err := p.GetByMemberId(characterId)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return g.IsLeader(characterId), nil
}
