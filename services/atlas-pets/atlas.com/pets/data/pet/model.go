package pet

type Model struct {
	id     uint32
	hunger uint32
	cash   bool
	life   uint32
	skills []SkillModel
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Hunger() uint32 {
	return m.hunger
}

func (m Model) Cash() bool {
	return m.cash
}

func (m Model) Life() uint32 {
	return m.life
}

func (m Model) Skills() []SkillModel {
	return m.skills
}

type ModelBuilder struct {
	id     uint32
	hunger uint32
	cash   bool
	life   uint32
	skills []SkillModel
}

func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

func (b *ModelBuilder) SetId(id uint32) *ModelBuilder {
	b.id = id
	return b
}

func (b *ModelBuilder) SetHunger(hunger uint32) *ModelBuilder {
	b.hunger = hunger
	return b
}

func (b *ModelBuilder) SetCash(cash bool) *ModelBuilder {
	b.cash = cash
	return b
}

func (b *ModelBuilder) SetLife(life uint32) *ModelBuilder {
	b.life = life
	return b
}

func (b *ModelBuilder) SetSkills(skills []SkillModel) *ModelBuilder {
	b.skills = skills
	return b
}

func (b *ModelBuilder) AddSkill(skill SkillModel) *ModelBuilder {
	if b.skills == nil {
		b.skills = []SkillModel{}
	}
	b.skills = append(b.skills, skill)
	return b
}

func (b *ModelBuilder) Build() Model {
	return Model{
		id:     b.id,
		hunger: b.hunger,
		cash:   b.cash,
		life:   b.life,
		skills: b.skills,
	}
}

type SkillModel struct {
	id          string
	increase    uint16
	probability uint16
}

func (m SkillModel) Id() string {
	return m.id
}

func (m SkillModel) Probability() uint16 {
	return m.probability
}

func (m SkillModel) Increase() uint16 {
	return m.increase
}

type SkillModelBuilder struct {
	id          string
	increase    uint16
	probability uint16
}

func NewSkillModelBuilder() *SkillModelBuilder {
	return &SkillModelBuilder{}
}

func (b *SkillModelBuilder) SetId(id string) *SkillModelBuilder {
	b.id = id
	return b
}

func (b *SkillModelBuilder) SetIncrease(increase uint16) *SkillModelBuilder {
	b.increase = increase
	return b
}

func (b *SkillModelBuilder) SetProbability(probability uint16) *SkillModelBuilder {
	b.probability = probability
	return b
}

func (b *SkillModelBuilder) Build() SkillModel {
	return SkillModel{
		id:          b.id,
		increase:    b.increase,
		probability: b.probability,
	}
}
