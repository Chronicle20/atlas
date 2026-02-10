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
	PinAttempts    int    `json:"pinAttempts"`
	PicAttempts    int    `json:"picAttempts"`
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
		PinAttempts:    m.PinAttempts(),
		PicAttempts:    m.PicAttempts(),
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

type PinAttemptInputRestModel struct {
	Id      string `json:"-"`
	Success bool   `json:"success"`
}

func (r PinAttemptInputRestModel) GetName() string {
	return "pin-attempts"
}

func (r PinAttemptInputRestModel) GetID() string {
	return r.Id
}

func (r *PinAttemptInputRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

type PinAttemptOutputRestModel struct {
	Id           string `json:"-"`
	Attempts     int    `json:"attempts"`
	LimitReached bool   `json:"limitReached"`
}

func (r PinAttemptOutputRestModel) GetName() string {
	return "pin-attempts"
}

func (r PinAttemptOutputRestModel) GetID() string {
	return r.Id
}

func (r *PinAttemptOutputRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

type PicAttemptInputRestModel struct {
	Id      string `json:"-"`
	Success bool   `json:"success"`
}

func (r PicAttemptInputRestModel) GetName() string {
	return "pic-attempts"
}

func (r PicAttemptInputRestModel) GetID() string {
	return r.Id
}

func (r *PicAttemptInputRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

type PicAttemptOutputRestModel struct {
	Id           string `json:"-"`
	Attempts     int    `json:"attempts"`
	LimitReached bool   `json:"limitReached"`
}

func (r PicAttemptOutputRestModel) GetName() string {
	return "pic-attempts"
}

func (r PicAttemptOutputRestModel) GetID() string {
	return r.Id
}

func (r *PicAttemptOutputRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return NewBuilder(uuid.Nil, rm.Name).
		SetId(rm.Id).
		SetPassword(rm.Password).
		SetPin(rm.Pin).
		SetPic(rm.Pic).
		SetPinAttempts(rm.PinAttempts).
		SetPicAttempts(rm.PicAttempts).
		SetState(State(rm.LoggedIn)).
		SetGender(rm.Gender).
		SetTOS(rm.TOS).
		Build()
}
