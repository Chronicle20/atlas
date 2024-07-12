package account

type Model struct {
	id       uint32
	name     string
	password string
	pin      string
	pic      string
	state    State
	gender   byte
	banned   bool
	tos      bool
}

func (a Model) Id() uint32 {
	return a.id
}

func (a Model) Name() string {
	return a.name
}

func (a Model) Password() string {
	return a.password
}

func (a Model) Banned() bool {
	return a.banned
}

func (a Model) State() State {
	return a.state
}

func (a Model) TOS() bool {
	return a.tos
}
