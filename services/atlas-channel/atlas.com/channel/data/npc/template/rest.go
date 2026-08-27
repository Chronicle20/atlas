package template

import "strconv"

type RestModel struct {
	Id        string `json:"-"`
	Name      string `json:"name"`
	TrunkPut  int32  `json:"trunk_put"`
	TrunkGet  int32  `json:"trunk_get"`
	Storebank bool   `json:"storebank"`
}

func (r RestModel) GetName() string {
	return "npcs"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// Transform is the inverse of Extract.
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:        strconv.Itoa(int(m.id)),
		Name:      m.name,
		TrunkPut:  m.trunkPut,
		TrunkGet:  m.trunkGet,
		Storebank: m.storebank,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	id, err := strconv.Atoi(rm.Id)
	if err != nil {
		return Model{}, err
	}

	return Model{
		id:        uint32(id),
		name:      rm.Name,
		trunkPut:  rm.TrunkPut,
		trunkGet:  rm.TrunkGet,
		storebank: rm.Storebank,
	}, nil
}
