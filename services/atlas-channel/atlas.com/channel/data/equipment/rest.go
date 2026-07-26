package equipment

import "strconv"

type RestModel struct {
	Id           uint32   `json:"-"`
	PetAbilities []string `json:"petAbilities,omitempty"`
}

func (r RestModel) GetName() string {
	return "statistics"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:           rm.Id,
		petAbilities: rm.PetAbilities,
	}, nil
}

type Model struct {
	id           uint32
	petAbilities []string
}

func (m Model) Id() uint32 {
	return m.id
}

// PetAbilities lists the equip's truthy pet-ability attributes (equip-family
// spelling: consumeHP, consumeMP, sweepForDrop, ...).
func (m Model) PetAbilities() []string {
	return m.petAbilities
}
