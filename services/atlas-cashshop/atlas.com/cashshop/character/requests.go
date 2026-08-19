package character

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters"
	ById     = Resource + "/%d"

	// EquipSlotExtensions is the atlas-character subresource ENABLE_EQUIP_SLOT
	// (task-240 task 23) reads and writes. Mirrors
	// services/atlas-character/atlas.com/character/equipslot/resource.go's
	// "/characters/{characterId}/equip-slot-extensions" path.
	EquipSlotExtensions = ById + "/equip-slot-extensions"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CHARACTERS")
}

func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ById, id))
}

func requestExtendEquipSlot(ctx context.Context, characterId uint32, slotIndex int16, days uint16, transactionId uuid.UUID) requests.Request[EquipSlotExtensionRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[EquipSlotExtensionRestModel](err)
	}
	body := ExtendEquipSlotInputRestModel{SlotIndex: slotIndex, Days: days, TransactionId: transactionId}
	return requests.PostRequest[EquipSlotExtensionRestModel](fmt.Sprintf(root+EquipSlotExtensions, characterId), body)
}
