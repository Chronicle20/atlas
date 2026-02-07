package reward

import "strconv"

type RestModel struct {
	Id         string `json:"-"`
	ItemId     uint32 `json:"itemId"`
	Quantity   uint32 `json:"quantity"`
	Tier       string `json:"tier"`
	GachaponId string `json:"gachaponId"`
}

func (r RestModel) GetName() string {
	return "gachapon-rewards"
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
		Id:         strconv.Itoa(int(m.ItemId())),
		ItemId:     m.ItemId(),
		Quantity:   m.Quantity(),
		Tier:       m.Tier(),
		GachaponId: m.GachaponId(),
	}, nil
}
