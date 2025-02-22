package guild

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func requestCreateProvider(m _map.Model, characterId uint32, name string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestCreateBody]{
		CharacterId: characterId,
		Type:        CommandTypeRequestCreate,
		Body: requestCreateBody{
			WorldId:   byte(m.WorldId()),
			ChannelId: byte(m.ChannelId()),
			MapId:     uint32(m.MapId()),
			Name:      name,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestInviteProvider(guildId uint32, characterId uint32, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestInviteBody]{
		CharacterId: characterId,
		Type:        CommandTypeRequestInvite,
		Body: requestInviteBody{
			GuildId:  guildId,
			TargetId: targetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func creationAgreementProvider(characterId uint32, agreed bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[creationAgreementBody]{
		CharacterId: characterId,
		Type:        CommandTypeCreationAgreement,
		Body: creationAgreementBody{
			Agreed: agreed,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeEmblemProvider(guildId uint32, characterId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[changeEmblemBody]{
		CharacterId: characterId,
		Type:        CommandTypeChangeEmblem,
		Body: changeEmblemBody{
			GuildId:             guildId,
			Logo:                logo,
			LogoColor:           logoColor,
			LogoBackground:      logoBackground,
			LogoBackgroundColor: logoBackgroundColor,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeNoticeProvider(guildId uint32, characterId uint32, notice string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[changeNoticeBody]{
		CharacterId: characterId,
		Type:        CommandTypeChangeNotice,
		Body: changeNoticeBody{
			GuildId: guildId,
			Notice:  notice,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeTitlesProvider(guildId uint32, characterId uint32, titles []string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[changeTitlesBody]{
		CharacterId: characterId,
		Type:        CommandTypeChangeTitles,
		Body: changeTitlesBody{
			GuildId: guildId,
			Titles:  titles,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeMemberTitleProvider(guildId uint32, characterId uint32, targetId uint32, newTitle byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[changeMemberTitleBody]{
		CharacterId: characterId,
		Type:        CommandTypeChangeMemberTitle,
		Body: changeMemberTitleBody{
			GuildId:  guildId,
			TargetId: targetId,
			Title:    newTitle,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func expelGuildProvider(guildId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[leaveBody]{
		CharacterId: characterId,
		Type:        CommandTypeLeave,
		Body: leaveBody{
			GuildId: guildId,
			Force:   true,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func leaveGuildProvider(guildId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[leaveBody]{
		CharacterId: characterId,
		Type:        CommandTypeLeave,
		Body: leaveBody{
			GuildId: guildId,
			Force:   false,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
