package consumable

import "strconv"

type SpecType string

const (
	SpecTypeHP         = SpecType("hp")
	SpecTypeMP         = SpecType("mp")
	SpecTypeHPRecovery = SpecType("hpR")
	SpecTypeMPRecovery = SpecType("mpR")
)

type RestModel struct {
	Id   uint32             `json:"-"`
	Spec map[SpecType]int32 `json:"spec"`
}

func (r RestModel) GetName() string {
	return "consumables"
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
// even though the consumable resource carries no relationships this client
// cares about (see libs/atlas-rest gotcha): a target struct must implement them
// or unmarshal errors whenever the upstream response includes a relationships
// block.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:   rm.Id,
		spec: rm.Spec,
	}, nil
}
