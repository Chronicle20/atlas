package character

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func loginEventProvider(messengerId uint32, worldId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &memberStatusEvent[memberLoginEventBody]{
		MessengerId:     messengerId,
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        EventMessengerMemberStatusTypeLogin,
		Body:        memberLoginEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func logoutEventProvider(messengerId uint32, worldId byte, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &memberStatusEvent[memberLogoutEventBody]{
		MessengerId:     messengerId,
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        EventMessengerMemberStatusTypeLogout,
		Body:        memberLogoutEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
