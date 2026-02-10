package account

import "strconv"

type RestModel struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	Password       string `json:"password"`
	Pin            string `json:"pin"`
	Pic            string `json:"pic"`
	PinAttempts    int    `json:"pinAttempts"`
	PicAttempts    int    `json:"picAttempts"`
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

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:             strconv.Itoa(int(m.id)),
		Name:           m.name,
		Pin:            m.pin,
		Pic:            m.pic,
		PinAttempts:    m.pinAttempts,
		PicAttempts:    m.picAttempts,
		LoggedIn:       byte(m.loggedIn),
		LastLogin:      m.lastLogin,
		Gender:         m.gender,
		Banned:         m.banned,
		TOS:            m.tos,
		Language:       m.language,
		Country:        m.country,
		CharacterSlots: m.characterSlots,
	}, nil
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
		SetPinAttempts(body.PinAttempts).
		SetPicAttempts(body.PicAttempts).
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
