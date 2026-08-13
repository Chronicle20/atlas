package cash

import (
	"strconv"
)

type SpecType string

const (
	SpecTypeInc        = SpecType("inc")
	SpecTypeIndexZero  = SpecType("0")
	SpecTypeIndexOne   = SpecType("1")
	SpecTypeIndexTwo   = SpecType("2")
	SpecTypeIndexThree = SpecType("3")
	SpecTypeIndexFour  = SpecType("4")
	SpecTypeIndexFive  = SpecType("5")
	SpecTypeIndexSix   = SpecType("6")
	SpecTypeIndexSeven = SpecType("7")
	SpecTypeIndexEight = SpecType("8")
	SpecTypeIndexNine  = SpecType("9")

	// Transformation-coupon properties (0530.img), mirroring atlas-data's
	// cash SpecType set (services/atlas-data/atlas.com/data/cash/rest.go).
	// `time` is the buff duration in MILLISECONDS, the unit atlas-buffs
	// expects — nothing on this path may rescale it.
	SpecTypeMorph = SpecType("morph")
	SpecTypeHp    = SpecType("hp")
	SpecTypeTime  = SpecType("time")
)

var SpecTypeIndexes = []SpecType{SpecTypeIndexZero, SpecTypeIndexOne, SpecTypeIndexTwo, SpecTypeIndexThree, SpecTypeIndexFour, SpecTypeIndexFive, SpecTypeIndexSix, SpecTypeIndexSeven, SpecTypeIndexEight, SpecTypeIndexNine}

type RestModel struct {
	Id          uint32             `json:"-"`
	SlotMax     uint32             `json:"slotMax"`
	Spec        map[SpecType]int32 `json:"spec"`
	PetSkills   []string           `json:"petSkills,omitempty"`
	PetSkillAdd bool               `json:"petSkillAdd,omitempty"`
}

func (r RestModel) GetName() string {
	return "cash_items"
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
		id:          rm.Id,
		slotMax:     rm.SlotMax,
		spec:        rm.Spec,
		petSkills:   rm.PetSkills,
		petSkillAdd: rm.PetSkillAdd,
	}, nil
}
