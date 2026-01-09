package storage

import (
	storage2 "atlas-saga-orchestrator/kafka/message/storage"
	storageCompartment "atlas-saga-orchestrator/kafka/message/storage/compartment"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"time"
)

func DepositCommandProvider(transactionId uuid.UUID, worldId byte, accountId uint32, slot int16, templateId uint32, expiration time.Time, referenceId uint32, referenceType string, referenceData storage2.ReferenceData) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &storage2.Command[storage2.DepositBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		AccountId:     accountId,
		Type:          storage2.CommandTypeDeposit,
		Body: storage2.DepositBody{
			Slot:          slot,
			TemplateId:    templateId,
			Expiration:    expiration,
			ReferenceId:   referenceId,
			ReferenceType: referenceType,
			ReferenceData: referenceData,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func WithdrawCommandProvider(transactionId uuid.UUID, worldId byte, accountId uint32, assetId uint32, quantity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &storage2.Command[storage2.WithdrawBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		AccountId:     accountId,
		Type:          storage2.CommandTypeWithdraw,
		Body: storage2.WithdrawBody{
			AssetId:  assetId,
			Quantity: quantity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func UpdateMesosCommandProvider(transactionId uuid.UUID, worldId byte, accountId uint32, mesos uint32, operation string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &storage2.Command[storage2.UpdateMesosBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		AccountId:     accountId,
		Type:          storage2.CommandTypeUpdateMesos,
		Body: storage2.UpdateMesosBody{
			Mesos:     mesos,
			Operation: operation,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DepositRollbackCommandProvider(transactionId uuid.UUID, worldId byte, accountId uint32, assetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &storage2.Command[storage2.DepositRollbackBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		AccountId:     accountId,
		Type:          storage2.CommandTypeDepositRollback,
		Body: storage2.DepositRollbackBody{
			AssetId: assetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ShowStorageCommandProvider(transactionId uuid.UUID, worldId byte, channelId byte, characterId uint32, npcId uint32, accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &storage2.ShowStorageCommand{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		CharacterId:   characterId,
		NpcId:         npcId,
		AccountId:     accountId,
		Type:          storage2.CommandTypeShowStorage,
	}
	return producer.SingleMessageProvider(key, value)
}

// AcceptCommandProvider creates an ACCEPT command for the storage compartment
func AcceptCommandProvider(transactionId uuid.UUID, worldId byte, accountId uint32, slot int16, templateId uint32, referenceId uint32, referenceType string, referenceData []byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &storageCompartment.Command[storageCompartment.AcceptCommandBody]{
		WorldId:   worldId,
		AccountId: accountId,
		Type:      storageCompartment.CommandAccept,
		Body: storageCompartment.AcceptCommandBody{
			TransactionId: transactionId,
			Slot:          slot,
			TemplateId:    templateId,
			ReferenceId:   referenceId,
			ReferenceType: referenceType,
			ReferenceData: referenceData,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ReleaseCommandProvider creates a RELEASE command for the storage compartment
func ReleaseCommandProvider(transactionId uuid.UUID, worldId byte, accountId uint32, assetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &storageCompartment.Command[storageCompartment.ReleaseCommandBody]{
		WorldId:   worldId,
		AccountId: accountId,
		Type:      storageCompartment.CommandRelease,
		Body: storageCompartment.ReleaseCommandBody{
			TransactionId: transactionId,
			AssetId:       assetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
