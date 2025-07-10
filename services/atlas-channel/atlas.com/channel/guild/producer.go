package guild

import (
	guild2 "atlas-channel/kafka/message/guild"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func RequestCreateProvider(m _map.Model, characterId uint32, name string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild2.Command[guild2.RequestCreateBody]{
		CharacterId: characterId,
		Type:        guild2.CommandTypeRequestCreate,
		Body: guild2.RequestCreateBody{
			WorldId:   m.WorldId(),
			ChannelId: m.ChannelId(),
			MapId:     m.MapId(),
			Name:      name,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestInviteProvider(guildId uint32, characterId uint32, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild2.Command[guild2.RequestInviteBody]{
		CharacterId: characterId,
		Type:        guild2.CommandTypeRequestInvite,
		Body: guild2.RequestInviteBody{
			GuildId:  guildId,
			TargetId: targetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CreationAgreementProvider(characterId uint32, agreed bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild2.Command[guild2.CreationAgreementBody]{
		CharacterId: characterId,
		Type:        guild2.CommandTypeCreationAgreement,
		Body: guild2.CreationAgreementBody{
			Agreed: agreed,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ChangeEmblemProvider(guildId uint32, characterId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild2.Command[guild2.ChangeEmblemBody]{
		CharacterId: characterId,
		Type:        guild2.CommandTypeChangeEmblem,
		Body: guild2.ChangeEmblemBody{
			GuildId:             guildId,
			Logo:                logo,
			LogoColor:           logoColor,
			LogoBackground:      logoBackground,
			LogoBackgroundColor: logoBackgroundColor,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ChangeNoticeProvider(guildId uint32, characterId uint32, notice string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild2.Command[guild2.ChangeNoticeBody]{
		CharacterId: characterId,
		Type:        guild2.CommandTypeChangeNotice,
		Body: guild2.ChangeNoticeBody{
			GuildId: guildId,
			Notice:  notice,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ChangeTitlesProvider(guildId uint32, characterId uint32, titles []string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild2.Command[guild2.ChangeTitlesBody]{
		CharacterId: characterId,
		Type:        guild2.CommandTypeChangeTitles,
		Body: guild2.ChangeTitlesBody{
			GuildId: guildId,
			Titles:  titles,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ChangeMemberTitleProvider(guildId uint32, characterId uint32, targetId uint32, newTitle byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild2.Command[guild2.ChangeMemberTitleBody]{
		CharacterId: characterId,
		Type:        guild2.CommandTypeChangeMemberTitle,
		Body: guild2.ChangeMemberTitleBody{
			GuildId:  guildId,
			TargetId: targetId,
			Title:    newTitle,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ExpelGuildProvider(guildId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild2.Command[guild2.LeaveBody]{
		CharacterId: characterId,
		Type:        guild2.CommandTypeLeave,
		Body: guild2.LeaveBody{
			GuildId: guildId,
			Force:   true,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func LeaveGuildProvider(guildId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &guild2.Command[guild2.LeaveBody]{
		CharacterId: characterId,
		Type:        guild2.CommandTypeLeave,
		Body: guild2.LeaveBody{
			GuildId: guildId,
			Force:   false,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
