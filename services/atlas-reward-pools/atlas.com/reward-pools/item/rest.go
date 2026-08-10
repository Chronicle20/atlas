package item

import "strconv"

type RestModel struct {
	Id          uint32 `json:"-"`
	GachaponId  string `json:"gachaponId"`
	ItemId      uint32 `json:"itemId"`
	Quantity    uint32 `json:"quantity"`
	Tier        string `json:"tier"`
	Weight      uint32 `json:"weight"`
	CommodityId uint32 `json:"commodityId"`
}

func (r RestModel) GetName() string {
	return "gachapon-items"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

// SetID is called unconditionally by api2go's Unmarshal, including for
// CREATE payloads, which carry no `data.id` because the row id is
// server-generated. An empty id therefore means "unset", not "malformed": it
// must leave Id at its zero value rather than fail the request, or every POST
// is rejected with a 400 before the handler runs. A non-empty id that isn't a
// number is still an error.
func (r *RestModel) SetID(idStr string) error {
	if idStr == "" {
		r.Id = 0
		return nil
	}
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:          m.Id(),
		GachaponId:  m.GachaponId(),
		ItemId:      m.ItemId(),
		Quantity:    m.Quantity(),
		Tier:        m.Tier(),
		Weight:      m.Weight(),
		CommodityId: m.CommodityId(),
	}, nil
}

type JSONModel struct {
	GachaponId  string `json:"gachaponId"`
	ItemId      uint32 `json:"itemId"`
	Quantity    uint32 `json:"quantity"`
	Tier        string `json:"tier"`
	Weight      uint32 `json:"weight"`
	CommodityId uint32 `json:"commodityId"`
}
