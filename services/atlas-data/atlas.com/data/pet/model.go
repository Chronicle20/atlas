package pet

type Model struct {
	Id     uint32       `json:"id"`
	Hungry uint32       `json:"hungry"`
	Cash   bool         `json:"cash"`
	Life   uint32       `json:"life"`
	Skills []SkillModel `json:"skills"`
}

func (m Model) GetId() uint32 {
	return m.Id
}

type SkillModel struct {
	Id          string `json:"id"`
	Increase    uint16 `json:"increase"`
	Probability uint16 `json:"probability"`
}
