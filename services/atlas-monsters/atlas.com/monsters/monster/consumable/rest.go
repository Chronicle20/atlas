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
