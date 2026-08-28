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
