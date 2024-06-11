package account

import "strconv"

type RestModel struct {
	Id             uint32 `json:"-"`
	Name           string `json:"name"`
	Password       string `json:"-"`
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
	return strconv.Itoa(int(r.Id))
}

func TransformAll(models []Model) []RestModel {
	rms := make([]RestModel, 0)
	for _, m := range models {
		rms = append(rms, Transform(m))
	}
	return rms
}

func Transform(model Model) RestModel {
	rm := RestModel{
		Id:             model.id,
		Name:           model.name,
		Password:       model.password,
		Pin:            "",
		Pic:            "",
		LoggedIn:       model.state,
		LastLogin:      0,
		Gender:         0,
		Banned:         false,
		TOS:            false,
		Language:       "en",
		Country:        "us",
		CharacterSlots: 4,
	}
	return rm
}
