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

func Transform(m Model) (RestModel, error) {
	factors := make([]FactorRestModel, 0, len(m.factors))
	for _, f := range m.factors {
		factors = append(factors, FactorRestModel{
			Source:     f.source,
			RateType:   f.rateType,
			Multiplier: f.multiplier,
		})
	}

	return RestModel{
		Id:           strconv.FormatUint(uint64(m.characterId), 10),
		ExpRate:      m.expRate,
		MesoRate:     m.mesoRate,
		ItemDropRate: m.itemDropRate,
		QuestExpRate: m.questExpRate,
		Factors:      factors,
	}, nil
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
