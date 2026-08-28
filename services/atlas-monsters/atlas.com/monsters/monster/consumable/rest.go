package consumable

import "strconv"

// RestModel is the narrow projection of atlas-data's consumable resource that
// the catch ladder needs. atlas-data returns many more fields; unmarshalling
// ignores the rest.
type RestModel struct {
	Id            uint32  `json:"-"`
	Create        uint32  `json:"create"`
	MonsterId     uint32  `json:"monsterId"`
	MonsterHP     uint32  `json:"monsterHP"`
	BridleProp    uint32  `json:"bridleProp"`
	BridlePropChg float64 `json:"bridlePropChg"`
}

func (r RestModel) GetName() string { return "consumables" }

func (r RestModel) GetID() string { return strconv.Itoa(int(r.Id)) }

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// SetToOneReferenceID is a no-op required by api2go's interface.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

// SetToManyReferenceIDs is a no-op required by api2go's interface.
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// Transform converts a Model to its RestModel wire shape.
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:            m.id,
		Create:        m.create,
		MonsterId:     m.monsterId,
		MonsterHP:     m.monsterHp,
		BridleProp:    m.bridleProp,
		BridlePropChg: m.bridlePropChg,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:            rm.Id,
		create:        rm.Create,
		monsterId:     rm.MonsterId,
		monsterHp:     rm.MonsterHP,
		bridleProp:    rm.BridleProp,
		bridlePropChg: rm.BridlePropChg,
	}, nil
}
