package account

import "strconv"

// RestModel no longer carries a flat CharacterSlots attribute: slots are
// per-(account, world), not per-account (task-246
// bug-b-type-must-add-a-slot.md). Callers that need a slot count now fetch
// CharacterSlotRestModel from the world-scoped sub-resource below instead.
type RestModel struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Password  string `json:"password"`
	Pin       string `json:"pin"`
	Pic       string `json:"pic"`
	BirthDate uint32 `json:"birthDate"`
	LoggedIn  byte   `json:"loggedIn"`
	LastLogin uint64 `json:"lastLogin"`
	Gender    byte   `json:"gender"`
	Banned    bool   `json:"banned"`
	TOS       bool   `json:"tos"`
	Language  string `json:"language"`
	Country   string `json:"country"`
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

// CharacterSlotRestModel mirrors atlas-account's own CharacterSlotRestModel
// (services/atlas-account/atlas.com/account/account/rest.go) and
// atlas-login's copy (services/atlas-login/atlas.com/login/account/rest.go),
// the request/response body of
// accounts/{accountId}/worlds/{worldId}/character-slots. It is used both as
// the GET response and as the (ignored) POST increment body.
type CharacterSlotRestModel struct {
	Id      string `json:"-"`
	WorldId byte   `json:"worldId"`
	Slots   int16  `json:"slots"`
}

func (r CharacterSlotRestModel) GetName() string {
	return "character-slots"
}

func (r CharacterSlotRestModel) GetID() string {
	return r.Id
}

func (r *CharacterSlotRestModel) SetID(idStr string) error {
	r.Id = idStr
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
		Build()
	return m, nil
}
