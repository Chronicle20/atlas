package party

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(partyId uint32) (Model, error) {
	return func(ctx context.Context) func(partyId uint32) (Model, error) {
		return func(partyId uint32) (Model, error) {
			return byIdProvider(l)(ctx)(partyId)()
		}
	}
}

func byIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(partyId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(partyId uint32) model.Provider[Model] {
		return func(partyId uint32) model.Provider[Model] {
			pp := requests.Provider[RestModel, Model](l, ctx)(requestById(partyId), Extract)
			mp := requests.SliceProvider[MemberRestModel, MemberModel](l, ctx)(requestMembers(partyId), ExtractMember, model.Filters[MemberModel]())
			return func() (Model, error) {
				p, err := pp()
				if err != nil {
					return Model{}, err
				}
				ms, err := mp()
				if err != nil {
					return Model{}, err
				}
				p.members = ms
				return p, nil
			}
		}
	}
}

func GetByMemberId(l logrus.FieldLogger) func(ctx context.Context) func(memberId uint32) (Model, error) {
	return func(ctx context.Context) func(memberId uint32) (Model, error) {
		return func(memberId uint32) (Model, error) {
			return model.First[Model](byMemberIdProvider(l)(ctx)(memberId), model.Filters[Model]())
		}
	}
}

func byMemberIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(memberId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(memberId uint32) model.Provider[[]Model] {
		return func(memberId uint32) model.Provider[[]Model] {
			pp := requests.SliceProvider[RestModel, Model](l, ctx)(requestByMemberId(memberId), Extract, model.Filters[Model]())
			return func() ([]Model, error) {
				ps, err := pp()
				if err != nil {
					return []Model{}, err
				}
				var results = make([]Model, 0)
				for _, p := range ps {
					ms, err := requests.SliceProvider[MemberRestModel, MemberModel](l, ctx)(requestMembers(p.Id()), ExtractMember, model.Filters[MemberModel]())()
					if err != nil {
						return []Model{}, err
					}
					p.members = ms
					results = append(results, p)
				}
				return results, nil
			}
		}
	}
}
