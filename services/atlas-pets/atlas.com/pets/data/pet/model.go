package pet

type Model struct {
	id          uint32
	hunger      uint32
	cash        bool
	life        uint32
	skills      []SkillModel
	reqPetLevel uint32
	reqItemId   uint32
	evolutions  []EvolutionModel
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

func (m Model) ReqPetLevel() uint32 {
	return m.reqPetLevel
}

func (m Model) ReqItemId() uint32 {
	return m.reqItemId
}

func (m Model) Evolutions() []EvolutionModel {
	return m.evolutions
}

func (m Model) IsEgg() bool {
	return len(m.evolutions) == 1 && m.reqItemId == 0 && m.reqPetLevel == 0
}

func (m Model) IsEvolvable() bool {
	return len(m.evolutions) > 0 && m.reqItemId != 0
}

type ModelBuilder struct {
	id          uint32
	hunger      uint32
	cash        bool
	life        uint32
	skills      []SkillModel
	reqPetLevel uint32
	reqItemId   uint32
	evolutions  []EvolutionModel
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

func (b *ModelBuilder) SetReqPetLevel(reqPetLevel uint32) *ModelBuilder {
	b.reqPetLevel = reqPetLevel
	return b
}

func (b *ModelBuilder) SetReqItemId(reqItemId uint32) *ModelBuilder {
	b.reqItemId = reqItemId
	return b
}

func (b *ModelBuilder) SetEvolutions(evolutions []EvolutionModel) *ModelBuilder {
	b.evolutions = evolutions
	return b
}

func (b *ModelBuilder) Build() Model {
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

type EvolutionModel struct {
	templateId  uint32
	probability uint32
}

func NewEvolutionModel(templateId uint32, probability uint32) EvolutionModel {
	return EvolutionModel{
		templateId:  templateId,
		probability: probability,
	}
}

func (e EvolutionModel) TemplateId() uint32 {
	return e.templateId
}

func (e EvolutionModel) Probability() uint32 {
	return e.probability
}
