package item

import (
	"strconv"
	"time"
)

type RestModel struct {
	Id          uint32 `json:"-"`
	CashId      int64  `json:"cashId,string"`
	TemplateId  uint32 `json:"templateId"`
	CommodityId uint32 `json:"commodityId"`
	Quantity    uint32 `json:"quantity"`
	Flag        uint16    `json:"flag"`
	PurchasedBy uint32    `json:"purchasedBy"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (r RestModel) GetName() string {
	return "items"
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
		Id:          m.Id(),
		CashId:      m.CashId(),
		TemplateId:  m.TemplateId(),
		CommodityId: m.CommodityId(),
		Quantity:    m.Quantity(),
		Flag:        m.Flag(),
		PurchasedBy: m.PurchasedBy(),
		CreatedAt:   m.CreatedAt(),
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:          rm.Id,
		cashId:      rm.CashId,
		templateId:  rm.TemplateId,
		commodityId: rm.CommodityId,
		quantity:    rm.Quantity,
		flag:        rm.Flag,
		purchasedBy: rm.PurchasedBy,
		createdAt:   rm.CreatedAt,
	}, nil
}
