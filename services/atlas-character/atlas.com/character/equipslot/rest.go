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

// ExtendInputRestModel is the POST write route's request body (task-240 task
// 23, R2 -- the write side task 22's InitResource deferred). SlotIndex
// carries the caller's already-resolved Atlas canonical position (R1); this
// route persists it as given and does not resolve or invent it. Days is the
// extension length in days.
type ExtendInputRestModel struct {
	Id        string `json:"-"`
	SlotIndex int16  `json:"slotIndex"`
	Days      uint16 `json:"days"`
}

func (r ExtendInputRestModel) GetName() string {
	return "equip-slot-extensions"
}

func (r ExtendInputRestModel) GetID() string {
	return r.Id
}

func (r *ExtendInputRestModel) SetID(id string) error {
	r.Id = id
	return nil
}
