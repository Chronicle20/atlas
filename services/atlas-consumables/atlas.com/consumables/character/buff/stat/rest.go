package stat

import "github.com/Chronicle20/atlas/libs/atlas-constants/character"

type RestModel struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

func Extract(rm RestModel) (Model, error) {
	return Model{Type: character.TemporaryStatType(rm.Type), Amount: rm.Amount}, nil
}
