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

func Extract(rm RestModel) (Model, error) {
	return Model{
		characterId:  rm.CharacterId,
		name:         rm.Name,
		jobId:        rm.JobId,
		level:        rm.Level,
		rank:         rm.Title,
		online:       rm.Online,
		allianceRank: rm.AllianceTitle,
	}, nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		CharacterId:   m.characterId,
		Name:          m.name,
		JobId:         m.jobId,
		Level:         m.level,
		Title:         m.rank,
		Online:        m.online,
		AllianceTitle: m.allianceRank,
	}, nil
}
