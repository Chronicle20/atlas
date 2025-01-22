package member

type Model struct {
	characterId   uint32
	name          string
	jobId         uint16
	level         byte
	title         byte
	online        bool
	allianceTitle byte
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) Name() string {
	return m.name
}

func (m Model) JobId() uint16 {
	return m.jobId
}

func (m Model) Level() byte {
	return m.level
}

func (m Model) Title() byte {
	return m.title
}

func (m Model) Online() bool {
	return m.online
}

func (m Model) AllianceTitle() byte {
	return m.allianceTitle
}
