package member

type RestModel struct {
	CharacterId   uint32 `json:"characterId"`
	Name          string `json:"name"`
	JobId         uint16 `json:"jobId"`
	Level         byte   `json:"level"`
	Title         byte   `json:"title"`
	Online        bool   `json:"online"`
	AllianceTitle byte   `json:"allianceTitle"`
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		CharacterId:   m.characterId,
		Name:          m.name,
		JobId:         m.jobId,
		Level:         m.level,
		Title:         m.title,
		Online:        m.online,
		AllianceTitle: m.allianceTitle,
	}, nil
}
