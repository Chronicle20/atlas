package transaction

import "time"

// RestModel is the JSON:API representation of a settled MTS transaction-history
// record (resource type "transactions"). It is read-only over REST: rows are
// written server-side at settle, never created through this surface.
type RestModel struct {
	Id             string    `json:"-"`
	WorldId        byte      `json:"worldId"`
	CharacterId    uint32    `json:"characterId"`
	CounterpartyId uint32    `json:"counterpartyId"`
	ItemId         uint32    `json:"itemId"`
	Quantity       uint32    `json:"quantity"`
	TotalPrice     uint32    `json:"totalPrice"`
	Kind           string    `json:"kind"`
	CreatedAt      time.Time `json:"createdAt"`
}

func (r RestModel) GetName() string {
	return "transactions"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:             m.Id().String(),
		WorldId:        byte(m.WorldId()),
		CharacterId:    m.CharacterId(),
		CounterpartyId: m.CounterpartyId(),
		ItemId:         m.ItemId(),
		Quantity:       m.Quantity(),
		TotalPrice:     m.TotalPrice(),
		Kind:           m.Kind(),
		CreatedAt:      m.CreatedAt(),
	}, nil
}
