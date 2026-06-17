package wish

import "time"

// RestModel is the JSON:API representation of a wish-list entry. The resource
// type is "wish-entries". On create only CharacterId and ItemId are read from
// the request attributes; Id and CreatedAt are server-assigned.
type RestModel struct {
	Id          string    `json:"-"`
	CharacterId uint32    `json:"characterId"`
	ItemId      uint32    `json:"itemId"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (r RestModel) GetName() string {
	return "wish-entries"
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
		Id:          m.Id().String(),
		CharacterId: m.CharacterId(),
		ItemId:      m.ItemId(),
		CreatedAt:   m.CreatedAt(),
	}, nil
}
