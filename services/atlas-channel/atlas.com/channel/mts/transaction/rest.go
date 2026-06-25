package transaction

import "time"

// RestModel mirrors atlas-mts's transaction.RestModel (the JSON:API
// "transactions" resource). Only the fields the channel renders into an ITCITEM
// are consumed here. Transactions carry no relationships block, so no api2go
// relationship stubs are required.
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

func (r RestModel) GetName() string { return "transactions" }
func (r RestModel) GetID() string   { return r.Id }

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Extract(r RestModel) (Model, error) {
	return Model{
		id:         r.Id,
		itemId:     r.ItemId,
		quantity:   r.Quantity,
		totalPrice: r.TotalPrice,
		kind:       r.Kind,
		createdAt:  r.CreatedAt,
	}, nil
}
