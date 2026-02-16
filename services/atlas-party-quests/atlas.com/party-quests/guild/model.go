package guild

type Model struct {
	id       uint32
	leaderId uint32
}

func (m Model) Id() uint32       { return m.id }
func (m Model) LeaderId() uint32 { return m.leaderId }
