package equipslot

import (
	"time"
)

// RestModel represents one of a character's currently-active equipped-inventory
// slot extensions. SlotIndex is the Atlas canonical equipped-inventory
// position (see derivation-equip-slot.md E1 / R1), not a wire value.
type RestModel struct {
	Id          string    `json:"-"`
	CharacterId uint32    `json:"characterId"`
	SlotIndex   int16     `json:"slotIndex"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r RestModel) GetName() string {
	return "equip-slot-extensions"
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:          m.Id().String(),
		CharacterId: m.CharacterId(),
		SlotIndex:   m.SlotIndex(),
		ExpiresAt:   m.ExpiresAt(),
	}, nil
}
