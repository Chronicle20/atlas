package account

type Model struct {
   id       uint32
   name     string
   password string
   state    byte
   banned   bool
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

func (a *Model) Banned() bool {
   return a.banned
}

func (a Model) State() byte {
   return a.state
}

type builder struct {
   id       uint32
   name     string
   password string
   state    byte
   banned   bool
}

func Builder(id uint32) *builder {
   return &builder{id: id}
}

func (a *builder) SetName(name string) *builder {
   a.name = name
   return a
}

func (a *builder) SetPassword(password string) *builder {
   a.password = password
   return a
}

func (a *builder) SetState(state byte) *builder {
   a.state = state
   return a
}

func (a *builder) Build() Model {
   return Model{
      id:       a.id,
      name:     a.name,
      password: a.password,
      state:    a.state,
      banned:   a.banned,
   }
}
