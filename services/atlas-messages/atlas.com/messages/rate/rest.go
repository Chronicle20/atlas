package rate

import (
	"strconv"
)

type RestModel struct {
	Id           string            `json:"-"`
	ExpRate      float64           `json:"expRate"`
	MesoRate     float64           `json:"mesoRate"`
	ItemDropRate float64           `json:"itemDropRate"`
	QuestExpRate float64           `json:"questExpRate"`
	Factors      []FactorRestModel `json:"factors,omitempty"`
}

func (r RestModel) GetName() string {
	return "rates"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

type FactorRestModel struct {
	Source     string  `json:"source"`
	RateType   string  `json:"rateType"`
	Multiplier float64 `json:"multiplier"`
}

func Extract(rm RestModel) (Model, error) {
	factors := make([]Factor, 0, len(rm.Factors))
	for _, f := range rm.Factors {
		factors = append(factors, Factor{
			source:     f.Source,
			rateType:   f.RateType,
			multiplier: f.Multiplier,
		})
	}

	characterId, _ := strconv.ParseUint(rm.Id, 10, 32)

	return Model{
		characterId:  uint32(characterId),
		expRate:      rm.ExpRate,
		mesoRate:     rm.MesoRate,
		itemDropRate: rm.ItemDropRate,
		questExpRate: rm.QuestExpRate,
		factors:      factors,
	}, nil
}
