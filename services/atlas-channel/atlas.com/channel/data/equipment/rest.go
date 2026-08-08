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

// SetToOneReferenceID / SetToManyReferenceIDs are required by api2go's unmarshal
// even though this client doesn't care about the equipment resource's
// relationships (see libs/atlas-rest gotcha): a target struct must implement
// them or unmarshal errors whenever the upstream response includes a
// relationships block.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:           rm.Id,
		petAbilities: rm.PetAbilities,
	}, nil
}
