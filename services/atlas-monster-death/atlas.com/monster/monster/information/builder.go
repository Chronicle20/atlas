package information

type Builder struct {
	hp         uint32
	experience uint32
	level      uint32
	name       string
}

func NewBuilder() *Builder {
	return &Builder{
		hp:         100,
		experience: 10,
	}
}

func (b *Builder) SetHp(hp uint32) *Builder {
	b.hp = hp
	return b
}

func (b *Builder) SetExperience(experience uint32) *Builder {
	b.experience = experience
	return b
}

func (b *Builder) SetLevel(level uint32) *Builder {
	b.level = level
	return b
}

func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

func (b *Builder) Build() (Model, error) {
	return Model{
		hp:         b.hp,
		experience: b.experience,
		level:      b.level,
		name:       b.name,
	}, nil
}
