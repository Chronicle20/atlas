package party

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func Create(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) error {
	return func(ctx context.Context) func(characterId uint32) error {
		return func(characterId uint32) error {
			l.Debugf("Character [%d] attempting to create a party.", characterId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(createCommandProvider(characterId))
		}
	}
}

func Leave(l logrus.FieldLogger) func(ctx context.Context) func(partyId uint32, characterId uint32) error {
	return func(ctx context.Context) func(partyId uint32, characterId uint32) error {
		return func(partyId uint32, characterId uint32) error {
			l.Debugf("Character [%d] attempting to leave party [%d].", characterId, partyId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(leaveCommandProvider(characterId, partyId, false))
		}
	}
}

func Expel(l logrus.FieldLogger) func(ctx context.Context) func(partyId uint32, characterId uint32, targetCharacterId uint32) error {
	return func(ctx context.Context) func(partyId uint32, characterId uint32, targetCharacterId uint32) error {
		return func(partyId uint32, characterId uint32, targetCharacterId uint32) error {
			l.Debugf("Character [%d] attempting to expel [%d] from party [%d].", characterId, targetCharacterId, partyId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(leaveCommandProvider(characterId, partyId, true))
		}
	}
}

func ChangeLeader(l logrus.FieldLogger) func(ctx context.Context) func(partyId uint32, characterId uint32, targetCharacterId uint32) error {
	return func(ctx context.Context) func(partyId uint32, characterId uint32, targetCharacterId uint32) error {
		return func(partyId uint32, characterId uint32, targetCharacterId uint32) error {
			l.Debugf("Character [%d] attempting to pass leadership to [%d] in party [%d].", characterId, targetCharacterId, partyId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(changeLeaderCommandProvider(characterId, partyId, targetCharacterId))
		}
	}
}

func RequestInvite(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, targetCharacterId uint32) error {
	return func(ctx context.Context) func(characterId uint32, targetCharacterId uint32) error {
		return func(characterId uint32, targetCharacterId uint32) error {
			l.Debugf("Character [%d] attempting to invite [%d] to a party.", characterId, targetCharacterId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestInviteCommandProvider(characterId, targetCharacterId))
		}
	}
}

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

func GetMemberIds(l logrus.FieldLogger) func(ctx context.Context) func(partyId uint32, filters []model.Filter[MemberModel]) model.Provider[[]uint32] {
	return func(ctx context.Context) func(partyId uint32, filters []model.Filter[MemberModel]) model.Provider[[]uint32] {
		return func(partyId uint32, filters []model.Filter[MemberModel]) model.Provider[[]uint32] {
			g, err := GetById(l)(ctx)(partyId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve party [%d].", partyId)
				return model.ErrorProvider[[]uint32](err)
			}
			ids := make([]uint32, 0)
			for _, m := range g.Members() {
				ok := true
				for _, f := range filters {
					if !f(m) {
						ok = false
					}
				}
				if ok {
					ids = append(ids, m.Id())
				}
			}
			return model.FixedProvider(ids)
		}
	}
}
