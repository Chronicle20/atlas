package wish

import "time"

// RestModel is the JSON:API representation of a wish-list entry. The resource
// type is "wish-entries". On create only CharacterId, WorldId, and ItemId are
// read from the request attributes; Id, Serial, and CreatedAt are server-
// assigned. WorldId/Serial back the channel's CANCEL_WISH serial -> wish
// resolution (the client echoes Serial as the ITCITEM's nITCSN).
type RestModel struct {
	Id          string    `json:"-"`
	WorldId     byte      `json:"worldId"`
	Serial      uint32    `json:"serial"`
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
		WorldId:     byte(m.WorldId()),
		Serial:      m.Serial(),
		CharacterId: m.CharacterId(),
		ItemId:      m.ItemId(),
		CreatedAt:   m.CreatedAt(),
	}, nil
}
