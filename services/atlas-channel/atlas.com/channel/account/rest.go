package account

import "strconv"

type RestModel struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	Password       string `json:"password"`
	Pin            string `json:"pin"`
	Pic            string `json:"pic"`
	LoggedIn       byte   `json:"loggedIn"`
	LastLogin      uint64 `json:"lastLogin"`
	Gender         byte   `json:"gender"`
	Banned         bool   `json:"banned"`
	TOS            bool   `json:"tos"`
	Language       string `json:"language"`
	Country        string `json:"country"`
	CharacterSlots int16  `json:"characterSlots"`
}

func (r RestModel) GetName() string {
	return "accounts"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Extract(body RestModel) (Model, error) {
	id, err := strconv.ParseUint(body.Id, 10, 32)
	if err != nil {
		return Model{}, err
	}
	m := NewBuilder().
		SetId(uint32(id)).
		SetName(body.Name).
		SetPassword(body.Password).
		SetPin(body.Pin).
		SetPic(body.Pic).
		SetLoggedIn(int(body.LoggedIn)).
		SetLastLogin(body.LastLogin).
		SetGender(body.Gender).
		SetBanned(body.Banned).
		SetTos(body.TOS).
		SetLanguage(body.Language).
		SetCountry(body.Country).
		SetCharacterSlots(body.CharacterSlots).
		Build()
	return m, nil
}
