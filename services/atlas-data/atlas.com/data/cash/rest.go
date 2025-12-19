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
)

var SpecTypeIndexes = []SpecType{SpecTypeIndexZero, SpecTypeIndexOne, SpecTypeIndexTwo, SpecTypeIndexThree, SpecTypeIndexFour, SpecTypeIndexFive, SpecTypeIndexSix, SpecTypeIndexSeven, SpecTypeIndexEight, SpecTypeIndexNine}

type RestModel struct {
	Id      uint32             `json:"-"`
	SlotMax uint32             `json:"slotMax"`
	Spec    map[SpecType]int32 `json:"spec"`
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
