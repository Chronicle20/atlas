package character

import (
	"context"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// EquipSlotExtensions is the atlas-character subresource ENABLE_EQUIP_SLOT
// (task-240 task 23) reads and writes. Mirrors
// services/atlas-character/atlas.com/character/equipslot/resource.go's
// "/characters/{characterId}/equip-slot-extensions" path.
const EquipSlotExtensions = ById + "/equip-slot-extensions"

// EquipSlotExtensionRestModel mirrors atlas-character's equipslot.RestModel
// (services/atlas-character/atlas.com/character/equipslot/rest.go): one
// character's equip-slot extension, read on GET and returned by the POST
// write this task adds. SlotIndex is the Atlas canonical equipped-inventory
// position (derivation-equip-slot.md E1 / R1) -- e.g. the pendant2 constant
// (libs/atlas-constants/inventory/slot) -- never a wire value.
type EquipSlotExtensionRestModel struct {
	Id          string    `json:"-"`
	CharacterId uint32    `json:"characterId"`
	SlotIndex   int16     `json:"slotIndex"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

func (r EquipSlotExtensionRestModel) GetName() string {
	return "equip-slot-extensions"
}

func (r EquipSlotExtensionRestModel) GetID() string {
	return r.Id
}

func (r *EquipSlotExtensionRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// ExtendEquipSlotInputRestModel is the POST body atlas-character's write
// route expects. SlotIndex carries the Atlas canonical position (R1) --
// the caller resolves it (e.g. via slot.GetSlotByType("pendant2")), atlas-
// character never invents it. Days is the extension's length; atlas-
// character converts it to a time.Duration.
type ExtendEquipSlotInputRestModel struct {
	Id        string `json:"-"`
	SlotIndex int16  `json:"slotIndex"`
	Days      uint16 `json:"days"`
}

func (r ExtendEquipSlotInputRestModel) GetName() string {
	return "equip-slot-extensions"
}

func (r ExtendEquipSlotInputRestModel) GetID() string {
	return r.Id
}

func (r *ExtendEquipSlotInputRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func requestExtendEquipSlot(ctx context.Context, characterId uint32, slotIndex int16, days uint16) requests.Request[EquipSlotExtensionRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[EquipSlotExtensionRestModel](err)
	}
	body := ExtendEquipSlotInputRestModel{SlotIndex: slotIndex, Days: days}
	return requests.PostRequest[EquipSlotExtensionRestModel](fmt.Sprintf(root+EquipSlotExtensions, characterId), body)
}

// ExtractEquipSlotExtension is the write route's response transformer --
// callers only need the resulting expiry, mirroring pet.Extract's shape
// (pet/rest.go) for a create-style POST.
func ExtractEquipSlotExtension(r EquipSlotExtensionRestModel) (time.Time, error) {
	return r.ExpiresAt, nil
}
