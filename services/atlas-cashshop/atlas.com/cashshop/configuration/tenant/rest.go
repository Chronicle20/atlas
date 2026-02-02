package tenant

import (
	"atlas-cashshop/configuration/tenant/cashshop"
)

type RestModel struct {
	Id       string             `json:"-"`
	CashShop cashshop.RestModel `json:"cashShop"`
}

func (r RestModel) GetName() string {
	return "tenants"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}
