package character

import (
	"atlas-messengers/kafka/message/character"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func loginEventProvider(messengerId uint32, worldId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &character.MemberStatusEvent[character.MemberLoginEventBody]{
		MessengerId:     messengerId,
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        character.EventMessengerMemberStatusTypeLogin,
		Body:        character.MemberLoginEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func logoutEventProvider(messengerId uint32, worldId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &character.MemberStatusEvent[character.MemberLogoutEventBody]{
		MessengerId:     messengerId,
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        character.EventMessengerMemberStatusTypeLogout,
		Body:        character.MemberLogoutEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
