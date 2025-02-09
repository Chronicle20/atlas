package skill

import "time"

type RestModel struct {
	Id          uint32    `json:"-"`
	Level       byte      `json:"level"`
	MasterLevel byte      `json:"masterLevel"`
	Expiration  time.Time `json:"expiration"`
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:          m.id,
		Level:       m.level,
		MasterLevel: m.masterLevel,
		Expiration:  m.expiration,
	}, nil
}
