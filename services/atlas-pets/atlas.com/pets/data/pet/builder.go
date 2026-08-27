package pet

type Builder struct {
	id          uint32
	hunger      uint32
	cash        bool
	life        uint32
	skills      []SkillModel
	reqPetLevel uint32
	reqItemId   uint32
	evolutions  []EvolutionModel
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetHunger(hunger uint32) *Builder {
	b.hunger = hunger
	return b
}

func (b *Builder) SetCash(cash bool) *Builder {
	b.cash = cash
	return b
}

func (b *Builder) SetLife(life uint32) *Builder {
	b.life = life
	return b
}

func (b *Builder) SetSkills(skills []SkillModel) *Builder {
	b.skills = skills
	return b
}

func (b *Builder) AddSkill(skill SkillModel) *Builder {
	if b.skills == nil {
		b.skills = []SkillModel{}
	}
	b.skills = append(b.skills, skill)
	return b
}

func (b *Builder) SetReqPetLevel(reqPetLevel uint32) *Builder {
	b.reqPetLevel = reqPetLevel
	return b
}

func (b *Builder) SetReqItemId(reqItemId uint32) *Builder {
	b.reqItemId = reqItemId
	return b
}

func (b *Builder) SetEvolutions(evolutions []EvolutionModel) *Builder {
	b.evolutions = evolutions
	return b
}

func (b *Builder) Build() Model {
	return Model{
		id:          b.id,
		hunger:      b.hunger,
		cash:        b.cash,
		life:        b.life,
		skills:      b.skills,
		reqPetLevel: b.reqPetLevel,
		reqItemId:   b.reqItemId,
		evolutions:  b.evolutions,
	}
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
