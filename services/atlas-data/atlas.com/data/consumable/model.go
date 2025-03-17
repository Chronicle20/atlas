package consumable

type SpecType string

const (
	SpecTypeHP                   = SpecType("hp")
	SpecTypeMP                   = SpecType("mp")
	SpecTypeHPRecovery           = SpecType("hpR")
	SpecTypeMPRecovery           = SpecType("mpR")
	SpecTypeMoveTo               = SpecType("moveTo")
	SpecTypeWeaponAttack         = SpecType("pad")
	SpecTypeMagicAttack          = SpecType("mad")
	SpecTypeWeaponDefense        = SpecType("pdd")
	SpecTypeMagicDefense         = SpecType("mdd")
	SpecTypeSpeed                = SpecType("speed")
	SpecTypeEvasion              = SpecType("eva")
	SpecTypeAccuracy             = SpecType("acc")
	SpecTypeJump                 = SpecType("jump")
	SpecTypeTime                 = SpecType("time")
	SpecTypeThaw                 = SpecType("thaw")
	SpecTypePoison               = SpecType("poison")
	SpecTypeDarkness             = SpecType("darkness")
	SpecTypeWeakness             = SpecType("weakness")
	SpecTypeSeal                 = SpecType("seal")
	SpecTypeCurse                = SpecType("curse")
	SpecTypeReturnMap            = SpecType("returnMapQR")
	SpecTypeIgnoreContinent      = SpecType("ignoreContinent")
	SpecTypeMorph                = SpecType("morph")
	SpecTypeRandomMoveInFieldSet = SpecType("randomMoveInFieldSet")
	SpecTypeExperienceBuff       = SpecType("expBuff")
	SpecTypeInc                  = SpecType("inc")
	SpecTypeOnlyPickup           = SpecType("onlyPickup")
)

type Model struct {
	Id              uint32             `json:"id"`
	TradeBlock      bool               `json:"trade_block"`
	Price           uint32             `json:"price"`
	UnitPrice       uint32             `json:"unit_price"`
	SlotMax         uint32             `json:"slot_max"`
	TimeLimited     bool               `json:"time_limited"`
	NotSale         bool               `json:"not_sale"`
	ReqLevel        uint32             `json:"req_level"`
	Quest           bool               `json:"quest"`
	Only            bool               `json:"only"`
	ConsumeOnPickup bool               `json:"consume_on_pickup"`
	Success         uint32             `json:"success"`
	Cursed          uint32             `json:"cursed"`
	Create          uint32             `json:"create"`
	MasterLevel     uint32             `json:"master_level"`
	ReqSkillLevel   uint32             `json:"req_skill_level"`
	TradeAvailable  bool               `json:"trade_available"`
	NoCancelMouse   bool               `json:"no_cancel_mouse"`
	Pquest          bool               `json:"pquest"`
	Left            int32              `json:"left"`
	Right           int32              `json:"right"`
	Top             int32              `json:"top"`
	Bottom          int32              `json:"bottom"`
	BridleMsgType   uint32             `json:"bridle_msg_type"`
	BridleProp      uint32             `json:"bridle_prop"`
	BridlePropChg   float64            `json:"bridle_prop_chg"`
	UseDelay        uint32             `json:"use_delay"`
	DelayMsg        string             `json:"delay_msg"`
	IncFatigue      int32              `json:"inc_fatigue"`
	Npc             uint32             `json:"npc"`
	Script          string             `json:"script"`
	RunOnPickup     bool               `json:"run_on_pickup"`
	MonsterBook     bool               `json:"monster_book"`
	MonsterId       uint32             `json:"monster_id"`
	BigSize         bool               `json:"big_size"`
	TargetBlock     bool               `json:"target_block"` // Fixed typo from "tragetBlock"
	Effect          string             `json:"effect"`
	MonsterHp       uint32             `json:"monster_hp"`
	WorldMsg        string             `json:"world_msg"`
	IncPDD          uint32             `json:"inc_pdd"`
	IncMDD          uint32             `json:"inc_mdd"`
	IncACC          uint32             `json:"inc_acc"`
	IncMHP          uint32             `json:"inc_mhp"`
	IncMMP          uint32             `json:"inc_mmp"`
	IncPAD          uint32             `json:"inc_pad"`
	IncMAD          uint32             `json:"inc_mad"`
	IncEVA          uint32             `json:"inc_eva"`
	IncLUK          uint32             `json:"inc_luk"`
	IncDEX          uint32             `json:"inc_dex"`
	IncINT          uint32             `json:"inc_int"`
	IncSTR          uint32             `json:"inc_str"`
	IncSpeed        uint32             `json:"inc_speed"`
	Spec            map[SpecType]int32 `json:"spec"`
	MonsterSummons  map[uint32]uint32  `json:"monster_summons"`
	Morphs          map[uint32]uint32  `json:"morphs"`
	Skills          []uint32           `json:"skills"`
	Rewards         []RewardModel      `json:"rewards"`
}

func (m Model) GetId() uint32 {
	return m.Id
}

type RewardModel struct {
	ItemId uint32 `json:"item_id"`
	Count  uint32 `json:"count"`
	Prob   uint32 `json:"prob"`
}
