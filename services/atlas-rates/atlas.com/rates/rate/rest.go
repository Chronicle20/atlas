package rate

import (
	"strconv"
)

// RestModel is the JSON:API representation of computed rates
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

// FactorRestModel is the JSON:API representation of a rate factor
type FactorRestModel struct {
	Source     string  `json:"source"`
	RateType   string  `json:"rateType"`
	Multiplier float64 `json:"multiplier"`
}

// TransformFactor converts a Factor to its REST model
func TransformFactor(f Factor) FactorRestModel {
	return FactorRestModel{
		Source:     f.source,
		RateType:   string(f.rateType),
		Multiplier: f.multiplier,
	}
}

// Transform converts Computed rates and factors to REST model
func Transform(characterId uint32, computed Computed, factors []Factor) RestModel {
	factorModels := make([]FactorRestModel, 0, len(factors))
	for _, f := range factors {
		factorModels = append(factorModels, TransformFactor(f))
	}

	return RestModel{
		Id:           strconv.FormatUint(uint64(characterId), 10),
		ExpRate:      computed.expRate,
		MesoRate:     computed.mesoRate,
		ItemDropRate: computed.itemDropRate,
		QuestExpRate: computed.questExpRate,
		Factors:      factorModels,
	}
}
