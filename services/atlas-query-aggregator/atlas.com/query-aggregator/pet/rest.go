package pet

import (
	"strconv"
	"time"
)

// RestModel represents the pet REST model returned from the pets service
type RestModel struct {
	Id         uint32    `json:"-"`
	CashId     uint64    `json:"cashId"`
	TemplateId uint32    `json:"templateId"`
	Name       string    `json:"name"`
	Level      byte      `json:"level"`
	Closeness  uint16    `json:"closeness"`
	Fullness   byte      `json:"fullness"`
	Expiration time.Time `json:"expiration"`
	OwnerId    uint32    `json:"ownerId"`
	Slot       int8      `json:"slot"`
	X          int16     `json:"x"`
	Y          int16     `json:"y"`
	Stance     byte      `json:"stance"`
	FH         int16     `json:"fh"`
	Flag       uint16    `json:"flag"`
	PurchaseBy uint32    `json:"purchaseBy"`
}

func (r RestModel) GetName() string {
	return "pets"
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

// Extract converts a RestModel to a domain Model
func Extract(rm RestModel) (Model, error) {
	return NewModel(rm.Id, rm.Slot), nil
}
