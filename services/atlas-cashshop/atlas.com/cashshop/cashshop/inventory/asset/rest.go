package asset

import (
	"strconv"
	"time"
)

type RestModel struct {
	Id            uint32    `json:"-"`
	CompartmentId string    `json:"compartmentId"`
	CashId        int64     `json:"cashId,string"`
	TemplateId    uint32    `json:"templateId"`
	CommodityId   uint32    `json:"commodityId"`
	Quantity      uint32    `json:"quantity"`
	Flag          uint16    `json:"flag"`
	PetId         uint32    `json:"petId"`
	PurchasedBy   uint32    `json:"purchasedBy"`
	Expiration    time.Time `json:"expiration"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (r RestModel) GetName() string {
	return "assets"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	if strId == "" {
		return nil
	}
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Transform(a Model) (RestModel, error) {
	return RestModel{
		Id:            a.Id(),
		CompartmentId: a.CompartmentId().String(),
		CashId:        a.CashId(),
		TemplateId:    a.TemplateId(),
		CommodityId:   a.CommodityId(),
		Quantity:      a.Quantity(),
		Flag:          a.Flag(),
		PetId:         a.PetId(),
		PurchasedBy:   a.PurchasedBy(),
		Expiration:    a.Expiration(),
		CreatedAt:     a.CreatedAt(),
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
		petId:       rm.PetId,
		purchasedBy: rm.PurchasedBy,
		expiration:  rm.Expiration,
		createdAt:   rm.CreatedAt,
	}, nil
}
