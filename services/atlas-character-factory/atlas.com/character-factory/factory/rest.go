package factory

import "strconv"

type RestModel struct {
	Id          uint32 `json:"-"`
	AccountId   uint32 `json:"accountId"`
	WorldId     byte   `json:"worldId"`
	Name        string `json:"name"`
	Gender      byte   `json:"gender"`
	JobIndex    uint32 `json:"jobIndex"`
	SubJobIndex uint32 `json:"subJobIndex"`
	Face        uint32 `json:"face"`
	Hair        uint32 `json:"hair"`
	HairColor   uint32 `json:"hairColor"`
	SkinColor   byte   `json:"skinColor"`
	Top         uint32 `json:"top"`
	Bottom      uint32 `json:"bottom"`
	Shoes       uint32 `json:"shoes"`
	Weapon      uint32 `json:"weapon"`
}

func (r RestModel) GetName() string {
	return "characters"
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

// CreateCharacterResponse represents the response for character creation requests
type CreateCharacterResponse struct {
	TransactionId string `json:"transactionId"`
}

func (r CreateCharacterResponse) GetName() string {
	return "characters"
}

func (r CreateCharacterResponse) GetID() string {
	return r.TransactionId
}

func (r *CreateCharacterResponse) SetID(strId string) error {
	r.TransactionId = strId
	return nil
}
