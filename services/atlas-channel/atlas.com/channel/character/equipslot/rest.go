package equipslot

import "time"

// RestModel is the read-side JSON:API projection of one of a character's
// active equip-slot extensions, as produced by atlas-character
// (services/atlas-character/atlas.com/character/equipslot/rest.go). SlotIndex
// is the Atlas canonical equipped-inventory position (see
// derivation-equip-slot.md E1 / R1), not a wire value -- atlas-channel only
// reads ExpiresAt out of it (task-240 task 23, R3/R4).
type RestModel struct {
	Id          string    `json:"-"`
	CharacterId uint32    `json:"characterId"`
	SlotIndex   int16     `json:"slotIndex"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

func (r RestModel) GetName() string {
	return "equip-slot-extensions"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// Transform is the identity transformer requests.SliceProvider needs --
// mirroring pendingchange.Transform's shape for a list resource that has no
// separate channel-side Model type.
func Transform(r RestModel) (RestModel, error) {
	return r, nil
}
