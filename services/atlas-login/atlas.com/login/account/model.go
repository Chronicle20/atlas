package account

type Model struct {
	id             uint32
	name           string
	password       string
	pin            string
	pic            string
	loggedIn       int
	lastLogin      uint64
	gender         byte
	banned         bool
	tos            bool
	language       string
	country        string
	characterSlots int16
}

func (a Model) Id() uint32 {
	return a.id
}

func (a Model) Name() string {
	return a.name
}

func (a Model) Gender() byte {
	return a.gender
}

func (a Model) PIC() string {
	return a.pic
}

func (a Model) CharacterSlots() int16 {
	return a.characterSlots
}

func (a Model) LoggedIn() int {
	return a.loggedIn
}

func (a Model) PIN() string {
	return a.pin
}

type builder struct {
	id             uint32
	name           string
	password       string
	pin            string
	pic            string
	loggedIn       int
	lastLogin      uint64
	gender         byte
	banned         bool
	tos            bool
	language       string
	country        string
	characterSlots int16
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

func (a *builder) SetCharacterSlots(characterSlots int16) *builder {
	a.characterSlots = characterSlots
	return a
}

func (a *builder) Build() Model {
	return Model{
		id:             a.id,
		name:           a.name,
		password:       a.password,
		pin:            a.pin,
		pic:            a.pic,
		loggedIn:       a.loggedIn,
		lastLogin:      a.lastLogin,
		gender:         a.gender,
		banned:         a.banned,
		tos:            a.tos,
		language:       a.language,
		country:        a.country,
		characterSlots: a.characterSlots,
	}
}
