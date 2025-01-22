package guild

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32) (Model, error) {
	return func(ctx context.Context) func(guildId uint32) (Model, error) {
		return func(guildId uint32) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(guildId), Extract)()
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

func RequestCreate(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, name string) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, name string) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, name string) error {
			l.Debugf("Character [%d] attempting to create guild [%s] in world [%d] channel [%d] map [%d].", characterId, name, worldId, channelId, mapId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestCreateProvider(worldId, channelId, mapId, characterId, name))
		}
	}
}

func CreationAgreement(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, agreed bool) error {
	return func(ctx context.Context) func(characterId uint32, agreed bool) error {
		return func(characterId uint32, agreed bool) error {
			l.Debugf("Character [%d] responded to guild creation agreement with [%t].", characterId, agreed)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(creationAgreementProvider(characterId, agreed))
		}
	}
}

func RequestEmblemUpdate(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, characterId uint32, logoBackground uint16, logoBackgroundColor byte, logo uint16, logoColor byte) error {
	return func(ctx context.Context) func(guildId uint32, characterId uint32, logoBackground uint16, logoBackgroundColor byte, logo uint16, logoColor byte) error {
		return func(guildId uint32, characterId uint32, logoBackground uint16, logoBackgroundColor byte, logo uint16, logoColor byte) error {
			l.Debugf("Character [%d] is attempting to change their guild emblem. Logo [%d], Logo Color [%d], Logo Background [%d], Logo Background Color [%d]", characterId, logo, logoColor, logoBackground, logoBackgroundColor)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(changeEmblemProvider(guildId, characterId, logo, logoColor, logoBackground, logoBackgroundColor))
		}
	}
}

func RequestNoticeUpdate(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, characterId uint32, notice string) error {
	return func(ctx context.Context) func(guildId uint32, characterId uint32, notice string) error {
		return func(guildId uint32, characterId uint32, notice string) error {
			l.Debugf("Character [%d] is attempting to set guild [%d] notice [%s].", characterId, guildId, notice)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(changeNoticeProvider(guildId, characterId, notice))
		}
	}
}
