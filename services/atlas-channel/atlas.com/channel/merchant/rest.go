package merchant

import "github.com/google/uuid"

type RestModel struct {
	Id           string `json:"-"`
	CharacterId  uint32 `json:"characterId"`
	ShopType     byte   `json:"shopType"`
	Title        string `json:"title"`
	MapId        uint32 `json:"mapId"`
	X            int16  `json:"x"`
	Y            int16  `json:"y"`
	PermitItemId uint32 `json:"permitItemId"`
	ListingCount int64  `json:"listingCount"`
}

func (r RestModel) GetName() string {
	return "merchants"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Extract(rm RestModel) (Model, error) {
	id, err := uuid.Parse(rm.Id)
	if err != nil {
		return Model{}, err
	}
	return Model{
		id:           id,
		characterId:  rm.CharacterId,
		shopType:     rm.ShopType,
		title:        rm.Title,
		x:            rm.X,
		y:            rm.Y,
		permitItemId: rm.PermitItemId,
		listingCount: rm.ListingCount,
	}, nil
}
