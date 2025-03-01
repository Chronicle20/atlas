package party

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
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
			return requests.Provider[RestModel, Model](l, ctx)(requestById(partyId), Extract)
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
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestByMemberId(memberId), Extract, model.Filters[Model]())
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

func MemberInMap(worldId world.Id, channelId channel.Id, mapId _map.Id) model.Filter[MemberModel] {
	return func(m MemberModel) bool {
		return m.online && m.worldId == worldId && m.channelId == channelId && m.mapId == mapId
	}
}
