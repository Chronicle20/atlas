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

// Builder is used to construct a Model instance
type Builder struct {
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

// NewBuilder creates a new Builder instance
func NewBuilder() *Builder {
	return &Builder{}
}

// SetId sets the id field
func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

// SetName sets the name field
func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

// SetPassword sets the password field
func (b *Builder) SetPassword(password string) *Builder {
	b.password = password
	return b
}

// SetPin sets the pin field
func (b *Builder) SetPin(pin string) *Builder {
	b.pin = pin
	return b
}

// SetPic sets the pic field
func (b *Builder) SetPic(pic string) *Builder {
	b.pic = pic
	return b
}

// SetLoggedIn sets the loggedIn field
func (b *Builder) SetLoggedIn(loggedIn int) *Builder {
	b.loggedIn = loggedIn
	return b
}

// SetLastLogin sets the lastLogin field
func (b *Builder) SetLastLogin(lastLogin uint64) *Builder {
	b.lastLogin = lastLogin
	return b
}

// SetGender sets the gender field
func (b *Builder) SetGender(gender byte) *Builder {
	b.gender = gender
	return b
}

// SetBanned sets the banned field
func (b *Builder) SetBanned(banned bool) *Builder {
	b.banned = banned
	return b
}

// SetTos sets the tos field
func (b *Builder) SetTos(tos bool) *Builder {
	b.tos = tos
	return b
}

// SetLanguage sets the language field
func (b *Builder) SetLanguage(language string) *Builder {
	b.language = language
	return b
}

// SetCountry sets the country field
func (b *Builder) SetCountry(country string) *Builder {
	b.country = country
	return b
}

// SetCharacterSlots sets the characterSlots field
func (b *Builder) SetCharacterSlots(characterSlots int16) *Builder {
	b.characterSlots = characterSlots
	return b
}

// Build creates a new Model instance with the Builder's values
func (b *Builder) Build() Model {
	return Model{
		id:             b.id,
		name:           b.name,
		password:       b.password,
		pin:            b.pin,
		pic:            b.pic,
		loggedIn:       b.loggedIn,
		lastLogin:      b.lastLogin,
		gender:         b.gender,
		banned:         b.banned,
		tos:            b.tos,
		language:       b.language,
		country:        b.country,
		characterSlots: b.characterSlots,
	}
}

// ToBuilder creates a Builder initialized with the Model's values
func (m Model) ToBuilder() *Builder {
	return NewBuilder().
		SetId(m.id).
		SetName(m.name).
		SetPassword(m.password).
		SetPin(m.pin).
		SetPic(m.pic).
		SetLoggedIn(m.loggedIn).
		SetLastLogin(m.lastLogin).
		SetGender(m.gender).
		SetBanned(m.banned).
		SetTos(m.tos).
		SetLanguage(m.language).
		SetCountry(m.country).
		SetCharacterSlots(m.characterSlots)
}
