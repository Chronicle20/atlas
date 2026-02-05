package storage

import (
	"atlas-channel/kafka/message/storage"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// ArrangeCommandProvider creates an ARRANGE command for the storage service
func ArrangeCommandProvider(worldId world.Id, accountId uint32, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &storage.Command[storage.ArrangeCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		AccountId:     accountId,
		Type:          storage.CommandTypeArrange,
		Body:          storage.ArrangeCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

// UpdateMesosCommandProvider creates an UPDATE_MESOS command for the storage service
func UpdateMesosCommandProvider(worldId world.Id, accountId uint32, transactionId uuid.UUID, mesos uint32, operation string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &storage.Command[storage.UpdateMesosCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		AccountId:     accountId,
		Type:          storage.CommandTypeUpdateMesos,
		Body: storage.UpdateMesosCommandBody{
			Mesos:     mesos,
			Operation: operation,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// CloseStorageCommandProvider creates a CLOSE_STORAGE command for the storage service
func CloseStorageCommandProvider(characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &storage.CloseStorageCommand{
		CharacterId: characterId,
		Type:        storage.CommandTypeCloseStorage,
	}
	return producer.SingleMessageProvider(key, value)
}
