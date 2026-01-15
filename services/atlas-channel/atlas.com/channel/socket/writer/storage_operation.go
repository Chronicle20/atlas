package writer

import (
	"atlas-channel/asset"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
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

func StorageOperationErrorInventoryFullBody(l logrus.FieldLogger) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(getStorageOperationMode(l)(options, StorageOperationModeErrorInventoryFull))
		return w.Bytes()
	}
}

func StorageOperationErrorNotEnoughMesoBody(l logrus.FieldLogger) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(getStorageOperationMode(l)(options, StorageOperationModeErrorNotEnoughMesos))
		return w.Bytes()
	}
}

func StorageOperationErrorOneOfAKindBody(l logrus.FieldLogger) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(getStorageOperationMode(l)(options, StorageOperationModeErrorOneOfAKind))
		return w.Bytes()
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
func StorageOperationUpdateAssetsForCompartmentBody(l logrus.FieldLogger, t tenant.Model) func(op StorageOperationMode, slots byte, inventoryType asset.InventoryType, assets []asset.Model[any]) BodyProducer {
	return func(op StorageOperationMode, slots byte, inventoryType asset.InventoryType, assets []asset.Model[any]) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getStorageOperationMode(l)(options, op))
			w.WriteByte(slots)

			// Set flag based on the affected compartment
			flags := inventoryTypeToFlag(inventoryType)
			w.WriteLong(uint64(flags))

			// Filter assets to only include those from the affected compartment
			var filteredAssets []asset.Model[any]
			for _, a := range assets {
				if a.InventoryType() == inventoryType {
					filteredAssets = append(filteredAssets, a)
				}
			}

			w.WriteByte(byte(len(filteredAssets)))
			_ = model.ForEachSlice(model.FixedProvider(filteredAssets), WriteAssetInfo(t)(true)(w))
			return w.Bytes()
		}
	}
}

func StorageOperationUpdateMesoBody(l logrus.FieldLogger) func(slots byte, meso uint32) BodyProducer {
	return func(slots byte, meso uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getStorageOperationMode(l)(options, StorageOperationModeUpdateMeso))
			w.WriteByte(slots)
			w.WriteLong(uint64(StorageFlagCurrency))
			w.WriteInt(meso)
			return w.Bytes()
		}
	}
}

func StorageOperationShowBody(l logrus.FieldLogger, t tenant.Model) func(npcId uint32, slots byte, meso uint32, assets []asset.Model[any]) BodyProducer {
	return func(npcId uint32, slots byte, meso uint32, assets []asset.Model[any]) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getStorageOperationMode(l)(options, StorageOperationModeShow))
			w.WriteInt(npcId)
			w.WriteByte(slots)
			w.WriteLong(uint64(StorageFlagCurrency | StorageFlagEquipment | StorageFlagConsumables | StorageFlagSetUp | StorageFlagEtc | StorageFlagCash))
			w.WriteInt(meso)
			w.WriteShort(0) // ??
			w.WriteByte(byte(len(assets)))
			_ = model.ForEachSlice(model.FixedProvider(assets), WriteAssetInfo(t)(true)(w))
			w.WriteShort(0)
			w.WriteByte(0)
			return w.Bytes()
		}
	}
}

func StorageOperationErrorMessageBody(l logrus.FieldLogger) func(message string) BodyProducer {
	return func(message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
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
