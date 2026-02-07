package account

import (
	"strconv"

	"github.com/google/uuid"
)

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
		Id:             m.Id(),
		Name:           m.Name(),
		Password:       m.Password(),
		Pin:            m.Pin(),
		Pic:            m.Pic(),
		LoggedIn:       byte(m.State()),
		LastLogin:      0,
		Gender:         m.Gender(),
		TOS:            m.TOS(),
		Language:       "en",
		Country:        "us",
		CharacterSlots: 4,
	}
	return rm, nil
}

func Extract(rm RestModel) (Model, error) {
	return NewBuilder(uuid.Nil, rm.Name).
		SetId(rm.Id).
		SetPassword(rm.Password).
		SetPin(rm.Pin).
		SetPic(rm.Pic).
		SetState(State(rm.LoggedIn)).
		SetGender(rm.Gender).
		SetTOS(rm.TOS).
		Build()
}
