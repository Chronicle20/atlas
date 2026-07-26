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

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:   rm.Id,
		spec: rm.Spec,
	}, nil
}

type Model struct {
	id   uint32
	spec map[SpecType]int32
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) GetSpec(specType SpecType) (int32, bool) {
	val, ok := m.spec[specType]
	return val, ok
}
