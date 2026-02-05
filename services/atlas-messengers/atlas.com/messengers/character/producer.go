package character

import (
	"atlas-messengers/kafka/message/character"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func loginEventProvider(transactionID uuid.UUID, messengerId uint32, worldId world.Id, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &character.MemberStatusEvent[character.MemberLoginEventBody]{
		TransactionId: transactionID,
		MessengerId:   messengerId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          character.EventMessengerMemberStatusTypeLogin,
		Body:          character.MemberLoginEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func logoutEventProvider(transactionID uuid.UUID, messengerId uint32, worldId world.Id, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &character.MemberStatusEvent[character.MemberLogoutEventBody]{
		TransactionId: transactionID,
		MessengerId:   messengerId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          character.EventMessengerMemberStatusTypeLogout,
		Body:          character.MemberLogoutEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
