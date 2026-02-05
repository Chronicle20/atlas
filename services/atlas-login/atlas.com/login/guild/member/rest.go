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
		characterId: rm.CharacterId,
	}, nil
}
