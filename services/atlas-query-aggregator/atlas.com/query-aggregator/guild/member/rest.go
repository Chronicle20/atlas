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

func Extract(rm RestModel) (Model, error) {
	return Model{
		characterId:  rm.CharacterId,
		name:         rm.Name,
		jobId:        rm.JobId,
		level:        rm.Level,
		rank:         rm.Rank,
		online:       rm.Online,
		allianceRank: rm.AllianceRank,
	}, nil
}
