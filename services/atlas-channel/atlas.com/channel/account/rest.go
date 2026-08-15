package account

import "strconv"

type RestModel struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	Password       string `json:"password"`
	Pin            string `json:"pin"`
	Pic            string `json:"pic"`
	BirthDate      uint32 `json:"birthDate"`
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

// PicAttemptInputRestModel is the POST body for
// accounts/{accountId}/pic-attempts, mirroring atlas-login's
// account.PicAttemptInputRestModel and atlas-account's own
// account.PicAttemptInputRestModel (services/atlas-account/atlas.com/account/account/rest.go).
type PicAttemptInputRestModel struct {
	Id        string `json:"-"`
	Success   bool   `json:"success"`
	IpAddress string `json:"ipAddress"`
	HWID      string `json:"hwid"`
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

// PicAttemptOutputRestModel is the response body from
// accounts/{accountId}/pic-attempts.
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
		SetBirthDate(body.BirthDate).
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
