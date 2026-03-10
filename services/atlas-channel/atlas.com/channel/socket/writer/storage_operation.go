package writer

import (
	"atlas-channel/asset"
	model2 "atlas-channel/socket/model"
	"context"

	storagepkt "github.com/Chronicle20/atlas-packet/storage"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

type StorageOperationMode string

type StorageFlag uint64

const (
	StorageOperation = "StorageOperation"

	StorageOperationModeRetrieveAssets       StorageOperationMode = "RETRIEVE_ASSETS"    // 9
	StorageOperationModeErrorInventoryFull   StorageOperationMode = "INVENTORY_FULL"     // 10
	StorageOperationModeErrorNotEnoughMesos  StorageOperationMode = "NOT_ENOUGH_MESOS"   // 11
	StorageOperationModeErrorOneOfAKind      StorageOperationMode = "ONE_OF_A_KIND"      // 12
	StorageOperationModeStoreAssets          StorageOperationMode = "STORE_ASSETS"       // 13 (another 'store' op at 15)
	StorageOperationModeErrorNotEnoughMesos2 StorageOperationMode = "NOT_ENOUGH_MESOS_2" // 16
	StorageOperationModeErrorUnknown         StorageOperationMode = "UNKNOWN"            // 17
	StorageOperationModeUpdateMeso           StorageOperationMode = "UPDATE_MESO"        // 19
	StorageOperationModeShow                 StorageOperationMode = "SHOW"               // 22
	StorageOperationModeErrorUnknown2        StorageOperationMode = "UNKNOWN_2"          // 23
	StorageOperationModeErrorMessage         StorageOperationMode = "ERROR_MESSAGE"      // 24

	StorageFlagCurrency    StorageFlag = 2
	StorageFlagEquipment   StorageFlag = 4
	StorageFlagConsumables StorageFlag = 8
	StorageFlagSetUp       StorageFlag = 16
	StorageFlagEtc         StorageFlag = 32
	StorageFlagCash        StorageFlag = 64
)

func StorageOperationErrorInventoryFullBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getStorageOperationMode(l)(options, StorageOperationModeErrorInventoryFull)
			return storagepkt.NewStorageErrorSimple(mode).Encode(l, ctx)(options)
		}
	}
}

func StorageOperationErrorNotEnoughMesoBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getStorageOperationMode(l)(options, StorageOperationModeErrorNotEnoughMesos)
			return storagepkt.NewStorageErrorSimple(mode).Encode(l, ctx)(options)
		}
	}
}

func StorageOperationErrorOneOfAKindBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getStorageOperationMode(l)(options, StorageOperationModeErrorOneOfAKind)
			return storagepkt.NewStorageErrorSimple(mode).Encode(l, ctx)(options)
		}
	}
}

// inventoryTypeToFlag converts an inventory type to the corresponding storage flag
func inventoryTypeToFlag(inventoryType asset.InventoryType) StorageFlag {
	switch inventoryType {
	case asset.InventoryTypeEquip:
		return StorageFlagEquipment
	case asset.InventoryTypeUse:
		return StorageFlagConsumables
	case asset.InventoryTypeSetup:
		return StorageFlagSetUp
	case asset.InventoryTypeEtc:
		return StorageFlagEtc
	case asset.InventoryTypeCash:
		return StorageFlagCash
	default:
		return 0
	}
}

// StorageOperationUpdateAssetsForCompartmentBody sends storage assets filtered by compartment type
// The flag is derived from the inventory type, and only assets of that type are sent
func StorageOperationUpdateAssetsForCompartmentBody(op StorageOperationMode, slots byte, inventoryType asset.InventoryType, assets []asset.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getStorageOperationMode(l)(options, op)
			flags := uint64(inventoryTypeToFlag(inventoryType))

			// Filter assets to only include those from the affected compartment
			var filteredAssets []asset.Model
			for _, a := range assets {
				if a.InventoryType() == inventoryType {
					filteredAssets = append(filteredAssets, a)
				}
			}

			// Pre-encode each asset
			assetEntryBytes := make([][]byte, len(filteredAssets))
			for i, a := range filteredAssets {
				am := model2.NewAsset(true, a)
				assetEntryBytes[i] = am.Encode(l, ctx)(options)
			}

			return storagepkt.NewStorageUpdateAssets(mode, slots, flags, assetEntryBytes).Encode(l, ctx)(options)
		}
	}
}

func StorageOperationUpdateMesoBody(slots byte, meso uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getStorageOperationMode(l)(options, StorageOperationModeUpdateMeso)
			return storagepkt.NewStorageUpdateMeso(mode, slots, meso).Encode(l, ctx)(options)
		}
	}
}

func StorageOperationShowBody(npcId uint32, slots byte, meso uint32, assets []asset.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getStorageOperationMode(l)(options, StorageOperationModeShow)
			flags := uint64(StorageFlagCurrency | StorageFlagEquipment | StorageFlagConsumables | StorageFlagSetUp | StorageFlagEtc | StorageFlagCash)

			// Pre-encode each asset
			assetEntryBytes := make([][]byte, len(assets))
			for i, a := range assets {
				am := model2.NewAsset(true, a)
				assetEntryBytes[i] = am.Encode(l, ctx)(options)
			}

			return storagepkt.NewStorageShow(mode, npcId, slots, flags, meso, assetEntryBytes).Encode(l, ctx)(options)
		}
	}
}

func StorageOperationErrorMessageBody(message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getStorageOperationMode(l)(options, StorageOperationModeErrorMessage)
			return storagepkt.NewStorageErrorMessage(mode, message).Encode(l, ctx)(options)
		}
	}
}

func getStorageOperationMode(l logrus.FieldLogger) func(options map[string]interface{}, key StorageOperationMode) byte {
	return func(options map[string]interface{}, key StorageOperationMode) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		op, ok := codes[string(key)].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}
		return byte(op)
	}
}
