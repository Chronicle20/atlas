package wallet

import (
	"github.com/google/uuid"
)

type RestModel struct {
	Id        uuid.UUID `json:"-"`
	AccountId uint32    `json:"accountId"`
	Credit    uint32    `json:"credit"`
	Points    uint32    `json:"points"`
	Prepaid   uint32    `json:"prepaid"`
}

func (r RestModel) GetName() string {
	return "wallets"
}

func (r RestModel) GetID() string {
	return r.Id.String()
}

func (r *RestModel) SetID(strId string) error {
	id, err := uuid.Parse(strId)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:        m.Id(),
		AccountId: m.AccountId(),
		Credit:    m.Credit(),
		Points:    m.Points(),
		Prepaid:   m.Prepaid(),
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:        rm.Id,
		accountId: rm.AccountId,
		credit:    rm.Credit,
		points:    rm.Points,
		prepaid:   rm.Prepaid,
	}, nil
}
