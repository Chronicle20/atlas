package wallet

import "github.com/google/uuid"

// RestModel is the cash-shop wallet payload. It is a flat (relationship-free)
// JSON:API resource, so no Unmarshal*Relations stubs are required. CurrencyType
// mapping (from the saga library): 1=credit, 2=points, 3=prepaid.
type RestModel struct {
	Id        uuid.UUID `json:"-"`
	AccountId uint32    `json:"accountId"`
	Credit    uint32    `json:"credit"`
	Points    uint32    `json:"points"`
	Prepaid   uint32    `json:"prepaid"`
}

func (r RestModel) GetName() string { return "wallets" }

func (r RestModel) GetID() string { return r.Id.String() }

func (r *RestModel) SetID(strId string) error {
	id, err := uuid.Parse(strId)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}
