package writer

import (
	"atlas-channel/asset"
	model2 "atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
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
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getStorageOperationMode(l)(options, StorageOperationModeErrorInventoryFull))
			return w.Bytes()
		}
	}
}

func StorageOperationErrorNotEnoughMesoBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getStorageOperationMode(l)(options, StorageOperationModeErrorNotEnoughMesos))
			return w.Bytes()
		}
	}
}

func StorageOperationErrorOneOfAKindBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getStorageOperationMode(l)(options, StorageOperationModeErrorOneOfAKind))
			return w.Bytes()
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
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getStorageOperationMode(l)(options, op))
			w.WriteByte(slots)

			// Set flag based on the affected compartment
			flags := inventoryTypeToFlag(inventoryType)
			w.WriteLong(uint64(flags))

			// Filter assets to only include those from the affected compartment
			var filteredAssets []asset.Model
			for _, a := range assets {
				if a.InventoryType() == inventoryType {
					filteredAssets = append(filteredAssets, a)
				}
			}

			w.WriteByte(byte(len(filteredAssets)))
			_ = model.ForEachSlice(model.FixedProvider(filteredAssets), model2.NewAssetWriter(l, ctx, options, w)(true))
			return w.Bytes()
		}
	}
}

func StorageOperationUpdateMesoBody(slots byte, meso uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getStorageOperationMode(l)(options, StorageOperationModeUpdateMeso))
			w.WriteByte(slots)
			w.WriteLong(uint64(StorageFlagCurrency))
			w.WriteInt(meso)
			return w.Bytes()
		}
	}
}

func StorageOperationShowBody(npcId uint32, slots byte, meso uint32, assets []asset.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getStorageOperationMode(l)(options, StorageOperationModeShow))
			w.WriteInt(npcId)
			w.WriteByte(slots)
			w.WriteLong(uint64(StorageFlagCurrency | StorageFlagEquipment | StorageFlagConsumables | StorageFlagSetUp | StorageFlagEtc | StorageFlagCash))
			w.WriteInt(meso)
			w.WriteShort(0) // ??
			w.WriteByte(byte(len(assets)))
			_ = model.ForEachSlice(model.FixedProvider(assets), model2.NewAssetWriter(l, ctx, options, w)(true))
			w.WriteShort(0)
			w.WriteByte(0)
			return w.Bytes()
		}
	}
}

func StorageOperationErrorMessageBody(message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getStorageOperationMode(l)(options, StorageOperationModeErrorMessage))
			w.WriteBool(true)
			w.WriteAsciiString(message)
			return w.Bytes()
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
