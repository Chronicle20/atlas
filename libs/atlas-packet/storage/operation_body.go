package storage

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/storage/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

type StorageOperationMode = string

type StorageFlag uint64

const (
	StorageOperationModeRetrieveAssets       StorageOperationMode = "RETRIEVE_ASSETS"
	StorageOperationModeErrorInventoryFull   StorageOperationMode = "INVENTORY_FULL"
	StorageOperationModeErrorNotEnoughMesos  StorageOperationMode = "NOT_ENOUGH_MESOS"
	StorageOperationModeErrorOneOfAKind      StorageOperationMode = "ONE_OF_A_KIND"
	StorageOperationModeStoreAssets          StorageOperationMode = "STORE_ASSETS"
	StorageOperationModeErrorNotEnoughMesos2 StorageOperationMode = "NOT_ENOUGH_MESOS_2"
	StorageOperationModeErrorUnknown         StorageOperationMode = "UNKNOWN"
	StorageOperationModeUpdateMeso           StorageOperationMode = "UPDATE_MESO"
	StorageOperationModeShow                 StorageOperationMode = "SHOW"
	StorageOperationModeErrorUnknown2        StorageOperationMode = "UNKNOWN_2"
	StorageOperationModeErrorMessage         StorageOperationMode = "ERROR_MESSAGE"

	StorageFlagCurrency    StorageFlag = 2
	StorageFlagEquipment   StorageFlag = 4
	StorageFlagConsumables StorageFlag = 8
	StorageFlagSetUp       StorageFlag = 16
	StorageFlagEtc         StorageFlag = 32
	StorageFlagCash        StorageFlag = 64

	StorageFlagAll = StorageFlagCurrency | StorageFlagEquipment | StorageFlagConsumables | StorageFlagSetUp | StorageFlagEtc | StorageFlagCash
)

func StorageOperationErrorInventoryFullBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", StorageOperationModeErrorInventoryFull, func(mode byte) packet.Encoder {
		return clientbound.NewStorageErrorSimple(mode)
	})
}

func StorageOperationErrorNotEnoughMesoBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", StorageOperationModeErrorNotEnoughMesos, func(mode byte) packet.Encoder {
		return clientbound.NewStorageErrorSimple(mode)
	})
}

func StorageOperationErrorOneOfAKindBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", StorageOperationModeErrorOneOfAKind, func(mode byte) packet.Encoder {
		return clientbound.NewStorageErrorSimple(mode)
	})
}

func StorageOperationUpdateAssetsBody(op StorageOperationMode, slots byte, flags uint64, assets []model.Asset) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", op, func(mode byte) packet.Encoder {
		return clientbound.NewStorageUpdateAssets(mode, slots, flags, assets)
	})
}

func StorageOperationUpdateMesoBody(slots byte, meso uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", StorageOperationModeUpdateMeso, func(mode byte) packet.Encoder {
		return clientbound.NewStorageUpdateMeso(mode, slots, meso)
	})
}

func StorageOperationShowBody(npcId uint32, slots byte, meso uint32, assets []model.Asset) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", StorageOperationModeShow, func(mode byte) packet.Encoder {
		return clientbound.NewStorageShow(mode, npcId, slots, uint64(StorageFlagAll), meso, assets)
	})
}

func StorageOperationErrorMessageBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", StorageOperationModeErrorMessage, func(mode byte) packet.Encoder {
		return clientbound.NewStorageErrorMessage(mode, message)
	})
}
