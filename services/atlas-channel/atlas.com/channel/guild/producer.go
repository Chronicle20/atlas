package guild

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func requestCreateProvider(worldId byte, channelId byte, mapId uint32, characterId uint32, name string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestCreateBody]{
		CharacterId: characterId,
		Type:        CommandTypeRequestCreate,
		Body: requestCreateBody{
			WorldId:   worldId,
			ChannelId: channelId,
			MapId:     mapId,
			Name:      name,
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
