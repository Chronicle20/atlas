package character

type Model struct {
	id    uint32
	name  string
	level byte
	jobId uint16
}

func (m Model) Name() string {
	return m.name
}

func (m Model) Level() byte {
	return m.level
}

func (m Model) JobId() uint16 {
	return m.jobId
}
