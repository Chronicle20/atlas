package commodity

import (
	"strconv"
)

type RestModel struct {
	Id       uint32 `json:"-"`
	ItemId   uint32 `json:"itemId"`
	Count    uint32 `json:"count"`
	Price    uint32 `json:"price"`
	Period   uint32 `json:"period"`
	Priority uint32 `json:"priority"`
	Gender   byte   `json:"gender"`
	OnSale   bool   `json:"onSale"`
}

func (r RestModel) GetName() string {
	return "commodities"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:       m.id,
		ItemId:   m.itemId,
		Count:    m.count,
		Price:    m.price,
		Period:   m.period,
		Priority: m.priority,
		Gender:   m.gender,
		OnSale:   m.onSale,
	}, nil
}
