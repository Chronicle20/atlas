package rates

import "strconv"

type RestModel struct {
	Id           string  `json:"-"`
	ExpRate      float64 `json:"expRate"`
	MesoRate     float64 `json:"mesoRate"`
	ItemDropRate float64 `json:"itemDropRate"`
	QuestExpRate float64 `json:"questExpRate"`
}

func (r RestModel) GetName() string {
	return "rates"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		expRate:      rm.ExpRate,
		mesoRate:     rm.MesoRate,
		itemDropRate: rm.ItemDropRate,
		questExpRate: rm.QuestExpRate,
	}, nil
}

// ExtractWithDefault returns default rates if extraction fails
func ExtractWithDefault(rm RestModel) Model {
	m, err := Extract(rm)
	if err != nil {
		return Default()
	}
	return m
}

// CharacterIdFromRestModel extracts characterId from the REST model ID
func CharacterIdFromRestModel(rm RestModel) (uint32, error) {
	id, err := strconv.ParseUint(rm.Id, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil
}
