package account

import "strconv"

type CreateRestModel struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Gender   byte   `json:"gender"`
}

func (r CreateRestModel) SetID(_ string) error {
	return nil
}

func (r CreateRestModel) GetName() string {
	return "accounts"
}

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

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	rm := RestModel{
		Id:             m.id,
		Name:           m.name,
		Password:       m.password,
		Pin:            m.pin,
		Pic:            m.pic,
		LoggedIn:       byte(m.state),
		LastLogin:      0,
		Gender:         m.gender,
		Banned:         false,
		TOS:            m.tos,
		Language:       "en",
		Country:        "us",
		CharacterSlots: 4,
	}
	return rm, nil
}

func Extract(rm RestModel) (Model, error) {
	m := Model{
		id:       rm.Id,
		name:     rm.Name,
		password: rm.Password,
		pin:      rm.Pin,
		pic:      rm.Pic,
		state:    State(rm.LoggedIn),
		gender:   rm.Gender,
		banned:   rm.Banned,
		tos:      rm.TOS,
	}
	return m, nil
}
