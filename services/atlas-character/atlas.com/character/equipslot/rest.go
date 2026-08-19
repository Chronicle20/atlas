package equipslot

import (
	"time"

	"github.com/google/uuid"
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

// TransformSlice maps a slice of domain Models to their REST projections.
// Returns the first transform error encountered, if any.
func TransformSlice(ms []Model) ([]RestModel, error) {
	rs := make([]RestModel, 0, len(ms))
	for _, m := range ms {
		r, err := Transform(m)
		if err != nil {
			return nil, err
		}
		rs = append(rs, r)
	}
	return rs, nil
}

// ExtendInputRestModel is the POST write route's request body (task-240 task
// 23, R2 -- the write side task 22's InitResource deferred). SlotIndex
// carries the caller's already-resolved Atlas canonical position (R1); this
// route persists it as given and does not resolve or invent it. Days is the
// extension length in days. TransactionId is atlas-cashshop's purchase
// idempotency key (task-240 task 24c): this route dedupes on it, via
// Extend/Entity.TransactionId, so a redelivered EXTEND_EQUIP_SLOT outbox
// command does not add days a second time. The zero UUID means "no dedupe
// key supplied" and always applies (matches every pre-task-24c caller).
type ExtendInputRestModel struct {
	Id            string    `json:"-"`
	SlotIndex     int16     `json:"slotIndex"`
	Days          uint16    `json:"days"`
	TransactionId uuid.UUID `json:"transactionId"`
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
