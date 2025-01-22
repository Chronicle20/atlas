package member

type Model struct {
	characterId  uint32
	name         string
	jobId        uint16
	level        byte
	rank         byte
	online       bool
	allianceRank byte
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

func (m Model) Rank() byte {
	return m.rank
}

func (m Model) Online() bool {
	return m.online
}

func (m Model) AllianceRank() byte {
	return m.allianceRank
}
