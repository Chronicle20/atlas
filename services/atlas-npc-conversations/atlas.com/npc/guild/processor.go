package guild

import (
	guild2 "atlas-npc-conversations/kafka/message/guild"
	"atlas-npc-conversations/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

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

func IsLeader(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) bool {
	return func(ctx context.Context) func(characterId uint32) bool {
		return func(characterId uint32) bool {
			g, _ := GetByMemberId(l)(ctx)(characterId)
			return g.LeaderId() == characterId
		}
	}
}

func HasGuild(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) bool {
	return func(ctx context.Context) func(characterId uint32) bool {
		return func(characterId uint32) bool {
			g, _ := GetByMemberId(l)(ctx)(characterId)
			return g.Id() != 0
		}
	}
}

func RequestName(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) error {
		return func(worldId byte, channelId byte, characterId uint32) error {
			l.Debugf("Requesting character [%d] input guild name for creation.", characterId)
			return producer.ProviderImpl(l)(ctx)(guild2.EnvCommandTopic)(requestNameProvider(worldId, channelId, characterId))
		}
	}
}

func RequestEmblem(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) error {
		return func(worldId byte, channelId byte, characterId uint32) error {
			l.Debugf("Requesting character [%d] input new guild emblem.", characterId)
			return producer.ProviderImpl(l)(ctx)(guild2.EnvCommandTopic)(requestEmblemProvider(worldId, channelId, characterId))
		}
	}
}

func RequestDisband(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) error {
		return func(worldId byte, channelId byte, characterId uint32) error {
			l.Debugf("Character [%d] attempting to disband guild.", characterId)
			return producer.ProviderImpl(l)(ctx)(guild2.EnvCommandTopic)(requestDisbandProvider(worldId, channelId, characterId))
		}
	}
}

func RequestCapacityIncrease(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32) error {
		return func(worldId byte, channelId byte, characterId uint32) error {
			l.Debugf("Character [%d] attempting to increase guild capacity.", characterId)
			return producer.ProviderImpl(l)(ctx)(guild2.EnvCommandTopic)(requestCapacityIncreaseProvider(worldId, channelId, characterId))
		}
	}
}
