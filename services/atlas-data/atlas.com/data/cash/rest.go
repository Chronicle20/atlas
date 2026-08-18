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
	Id          uint32 `json:"-"`
	SlotMax     uint32 `json:"slotMax"`
	ProtectTime uint32 `json:"protectTime,omitempty"`
	// AddTime is info/addTime in SECONDS — the expiration grant of an
	// item-expiration extender (Magical Sandglass, classification 550). The
	// client multiplies it by 10^7 into FILETIME 100ns units
	// (CDraggableItem::ModifyEquipItem, gms_v83 @0x4F4BB7), which is what
	// fixes the unit as seconds.
	AddTime uint32 `json:"addTime,omitempty"`
	// MaxDays is info/maxDays in DAYS — the ceiling, anchored to now, past
	// which an extender may not push a target's expiration.
	MaxDays uint32 `json:"maxDays,omitempty"`
	// Life is info/life in DAYS — the lifespan a pet-revival item (Water of
	// Life, classification 518) grants. Same WZ node and same unit as
	// pet/reader.go's Life; 0518.img carries life but no maxDays, which is
	// why the expiration-extender's maxDays ceiling cannot serve this flow.
	Life            uint32 `json:"life,omitempty"`
	Meso            uint32 `json:"meso,omitempty"` // 0520 meso sacks: info/meso award amount
	StateChangeItem uint32 `json:"stateChangeItem,omitempty"`
	// Npc is the WZ info/npc value: the NPC template a remote-merchant cash
	// item (classification 545) opens. 0 when the item targets no NPC.
	Npc         uint32             `json:"npc,omitempty"`
	BgmPath     string             `json:"bgmPath,omitempty"`
	Spec        map[SpecType]int32 `json:"spec"`
	TimeWindows []TimeWindow       `json:"timeWindows,omitempty"` // Active time windows from info/time
	PetSkills   []string           `json:"petSkills,omitempty"`
	PetSkillAdd bool               `json:"petSkillAdd,omitempty"`
	TradeBlock  bool               `json:"tradeBlock"`
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
