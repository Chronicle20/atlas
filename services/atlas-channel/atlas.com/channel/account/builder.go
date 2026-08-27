package account

type builder struct {
	id        uint32
	name      string
	password  string
	pin       string
	pic       string
	birthDate uint32
	loggedIn  int
	lastLogin uint64
	gender    byte
	banned    bool
	tos       bool
	language  string
	country   string
}

func NewBuilder() *builder {
	return &builder{}
}

func (a *builder) SetId(id uint32) *builder {
	a.id = id
	return a
}

func (a *builder) SetName(name string) *builder {
	a.name = name
	return a
}

func (a *builder) SetPassword(password string) *builder {
	a.password = password
	return a
}

func (a *builder) SetPin(pin string) *builder {
	a.pin = pin
	return a
}

func (a *builder) SetPic(pic string) *builder {
	a.pic = pic
	return a
}

func (a *builder) SetBirthDate(birthDate uint32) *builder {
	a.birthDate = birthDate
	return a
}

func (a *builder) SetLoggedIn(loggedIn int) *builder {
	a.loggedIn = loggedIn
	return a
}

func (a *builder) SetLastLogin(lastLogin uint64) *builder {
	a.lastLogin = lastLogin
	return a
}

func (a *builder) SetGender(gender byte) *builder {
	a.gender = gender
	return a
}

func (a *builder) SetBanned(banned bool) *builder {
	a.banned = banned
	return a
}

func (a *builder) SetTos(tos bool) *builder {
	a.tos = tos
	return a
}

func (a *builder) SetLanguage(language string) *builder {
	a.language = language
	return a
}

func (a *builder) SetCountry(country string) *builder {
	a.country = country
	return a
}

func (a *builder) Build() Model {
	return Model{
		id:        a.id,
		name:      a.name,
		password:  a.password,
		pin:       a.pin,
		pic:       a.pic,
		birthDate: a.birthDate,
		loggedIn:  a.loggedIn,
		lastLogin: a.lastLogin,
		gender:    a.gender,
		banned:    a.banned,
		tos:       a.tos,
		language:  a.language,
		country:   a.country,
	}
}
