package item

import "strconv"

type RestModel struct {
	Id          uint32 `json:"-"`
	CashId      uint64 `json:"cashId"`
	TemplateId  uint32 `json:"templateId"`
	Quantity    uint32 `json:"quantity"`
	Flag        uint16 `json:"flag"`
	PurchasedBy uint32 `json:"purchasedBy"`
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
		Id:          m.id,
		CashId:      m.cashId,
		TemplateId:  m.templateId,
		Quantity:    m.quantity,
		Flag:        m.flag,
		PurchasedBy: m.purchasedBy,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:          rm.Id,
		cashId:      rm.CashId,
		templateId:  rm.TemplateId,
		quantity:    rm.Quantity,
		flag:        rm.Flag,
		purchasedBy: rm.PurchasedBy,
	}, nil
}
