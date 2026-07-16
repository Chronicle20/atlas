package searchcount

import "strconv"

type RestModel struct {
	Id     string `json:"-"`
	ItemId uint32 `json:"itemId"`
	Count  uint64 `json:"count"`
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r RestModel) GetName() string {
	return "shop-search-counts"
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:     strconv.FormatUint(uint64(m.ItemId()), 10),
		ItemId: m.ItemId(),
		Count:  m.Count(),
	}, nil
}
