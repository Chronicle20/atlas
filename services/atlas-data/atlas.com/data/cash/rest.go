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
	SpecTypeTime = SpecType("time") // Duration from spec node; raw WZ units (0530 morph coupons: milliseconds)
	// Transformation-coupon properties (0530.img): the Morph.wz creature id and
	// the flat HP heal. Both are omit-when-zero in the reader, so downstream
	// "absent or zero" collapses to a single `ok && val > 0` test.
	SpecTypeMorph = SpecType("morph")
	SpecTypeHp    = SpecType("hp")
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
	ProtectTime     uint32             `json:"protectTime,omitempty"`
	Meso            uint32             `json:"meso,omitempty"` // 0520 meso sacks: info/meso award amount
	StateChangeItem uint32             `json:"stateChangeItem,omitempty"`
	BgmPath         string             `json:"bgmPath,omitempty"`
	Spec            map[SpecType]int32 `json:"spec"`
	TimeWindows     []TimeWindow       `json:"timeWindows,omitempty"` // Active time windows from info/time
	PetSkills       []string           `json:"petSkills,omitempty"`
	PetSkillAdd     bool               `json:"petSkillAdd,omitempty"`
	TradeBlock      bool               `json:"tradeBlock"`
	// TradeAvailable is WZ info/tradeAvailable; see equipment/rest.go.
	TradeAvailable int32 `json:"tradeAvailable"`
	// Karma is WZ info/karma — the SCISSORS' OWN karma type, read by
	// CItemInfo::RegisterKarmaScissorsItem (gms_v95 @0x5A1120) into
	// KARMASCISSORSITEM.nKarmaType. Parsed for every cash item and left 0 for
	// non-scissors: absence already yields 0, so no classification filter is
	// needed. The v83 corpus carries no `karma` node at all, which is why the
	// eligibility predicate treats 0 as "untyped scissors" (design §3.2).
	Karma int32 `json:"karma"`
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
