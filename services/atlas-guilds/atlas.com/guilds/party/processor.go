package party

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(partyId uint32) (Model, error)
	GetByMemberId(memberId uint32) (Model, error)
	ByIdProvider(partyId uint32) model.Provider[Model]
	ByMemberIdProvider(memberId uint32) model.Provider[[]Model]
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

func (p *ProcessorImpl) GetById(partyId uint32) (Model, error) {
	return p.ByIdProvider(partyId)()
}

func (p *ProcessorImpl) ByIdProvider(partyId uint32) model.Provider[Model] {
	pp := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(partyId), Extract)
	mp := requests.SliceProvider[MemberRestModel, MemberModel](p.l, p.ctx)(requestMembers(partyId), ExtractMember, model.Filters[MemberModel]())
	return func() (Model, error) {
		pa, err := pp()
		if err != nil {
			return Model{}, err
		}
		ms, err := mp()
		if err != nil {
			return Model{}, err
		}
		pa.members = ms
		return pa, nil
	}
}

func (p *ProcessorImpl) GetByMemberId(memberId uint32) (Model, error) {
	return model.First[Model](p.ByMemberIdProvider(memberId), model.Filters[Model]())
}

func (p *ProcessorImpl) ByMemberIdProvider(memberId uint32) model.Provider[[]Model] {
	pp := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByMemberId(memberId), Extract, model.Filters[Model]())
	return func() ([]Model, error) {
		ps, err := pp()
		if err != nil {
			return []Model{}, err
		}
		var results = make([]Model, 0)
		for _, pa := range ps {
			ms, err := requests.SliceProvider[MemberRestModel, MemberModel](p.l, p.ctx)(requestMembers(pa.Id()), ExtractMember, model.Filters[MemberModel]())()
			if err != nil {
				return []Model{}, err
			}
			pa.members = ms
			results = append(results, pa)
		}
		return results, nil
	}
}
