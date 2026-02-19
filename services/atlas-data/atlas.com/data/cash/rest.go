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
	// Rate coupon properties (EXP coupons in 0521.img, Drop coupons in 0536.img)
	SpecTypeRate = SpecType("rate") // Rate multiplier from info node (e.g., 2 for 2x)
	SpecTypeExpR = SpecType("expR") // EXP rate value from spec node
	SpecTypeDrpR = SpecType("drpR") // Drop rate value from spec node
	SpecTypeTime = SpecType("time") // Duration in minutes from spec node
)

var SpecTypeIndexes = []SpecType{SpecTypeIndexZero, SpecTypeIndexOne, SpecTypeIndexTwo, SpecTypeIndexThree, SpecTypeIndexFour, SpecTypeIndexFive, SpecTypeIndexSix, SpecTypeIndexSeven, SpecTypeIndexEight, SpecTypeIndexNine}

// TimeWindow represents an active time window for a coupon (e.g., "MON:18-20")
type TimeWindow struct {
	Day       string `json:"day"`       // Day of week: MON, TUE, WED, THU, FRI, SAT, SUN, HOL
	StartHour int    `json:"startHour"` // Start hour (0-23)
	EndHour   int    `json:"endHour"`   // End hour (1-24, where 24 means midnight)
}

type RestModel struct {
	Id              uint32             `json:"-"`
	SlotMax         uint32             `json:"slotMax"`
	StateChangeItem uint32             `json:"stateChangeItem,omitempty"`
	BgmPath         string             `json:"bgmPath,omitempty"`
	Spec            map[SpecType]int32 `json:"spec"`
	TimeWindows     []TimeWindow       `json:"timeWindows,omitempty"` // Active time windows from info/time
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
