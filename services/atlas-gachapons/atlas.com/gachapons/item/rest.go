package item

import "strconv"

type RestModel struct {
	Id         uint32 `json:"-"`
	GachaponId string `json:"gachaponId"`
	ItemId     uint32 `json:"itemId"`
	Quantity   uint32 `json:"quantity"`
	Tier       string `json:"tier"`
}

func (r RestModel) GetName() string {
	return "gachapon-items"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:         m.Id(),
		GachaponId: m.GachaponId(),
		ItemId:     m.ItemId(),
		Quantity:   m.Quantity(),
		Tier:       m.Tier(),
	}, nil
}

type JSONModel struct {
	GachaponId string `json:"gachaponId"`
	ItemId     uint32 `json:"itemId"`
	Quantity   uint32 `json:"quantity"`
	Tier       string `json:"tier"`
}
