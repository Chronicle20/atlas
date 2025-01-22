package member

type RestModel struct {
	CharacterId  uint32 `json:"characterId"`
	Name         string `json:"name"`
	JobId        uint16 `json:"jobId"`
	Level        byte   `json:"level"`
	Rank         byte   `json:"rank"`
	Online       bool   `json:"online"`
	AllianceRank byte   `json:"allianceRank"`
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		CharacterId:  m.characterId,
		Name:         m.name,
		JobId:        m.jobId,
		Level:        m.level,
		Rank:         m.rank,
		Online:       m.online,
		AllianceRank: m.allianceRank,
	}, nil
}
