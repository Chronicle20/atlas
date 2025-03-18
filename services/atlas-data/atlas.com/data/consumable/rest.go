package consumable

import (
	"strconv"
)

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

type RestModel struct {
	Id              uint32             `json:"-"`
	TradeBlock      bool               `json:"tradeBlock"`
	Price           uint32             `json:"price"`
	UnitPrice       uint32             `json:"unitPrice"`
	SlotMax         uint32             `json:"slotMax"`
	TimeLimited     bool               `json:"timeLimited"`
	NotSale         bool               `json:"notSale"`
	ReqLevel        uint32             `json:"reqLevel"`
	Quest           bool               `json:"quest"`
	Only            bool               `json:"only"`
	ConsumeOnPickup bool               `json:"consumeOnPickup"`
	Success         uint32             `json:"success"`
	Cursed          uint32             `json:"cursed"`
	Create          uint32             `json:"create"`
	MasterLevel     uint32             `json:"masterLevel"`
	ReqSkillLevel   uint32             `json:"reqSkillLevel"`
	TradeAvailable  bool               `json:"tradeAvailable"`
	NoCancelMouse   bool               `json:"noCancelMouse"`
	Pquest          bool               `json:"pquest"`
	Left            int32              `json:"left"`
	Right           int32              `json:"right"`
	Top             int32              `json:"top"`
	Bottom          int32              `json:"bottom"`
	BridleMsgType   uint32             `json:"bridleMsgType"`
	BridleProp      uint32             `json:"bridleProp"`
	BridlePropChg   float64            `json:"bridlePropChg"`
	UseDelay        uint32             `json:"useDelay"`
	DelayMsg        string             `json:"delayMsg"`
	IncFatigue      int32              `json:"incFatigue"`
	Npc             uint32             `json:"npc"`
	Script          string             `json:"script"`
	RunOnPickup     bool               `json:"runOnPickup"`
	MonsterBook     bool               `json:"monsterBook"`
	MonsterId       uint32             `json:"monsterId"`
	BigSize         bool               `json:"bigSize"`
	TargetBlock     bool               `json:"targetBlock"`
	Effect          string             `json:"effect"`
	MonsterHP       uint32             `json:"monsterHP"`
	WorldMsg        string             `json:"worldMsg"`
	IncreasePDD     uint32             `json:"increasePDD"`
	IncreaseMDD     uint32             `json:"increaseMDD"`
	IncreaseACC     uint32             `json:"increaseACC"`
	IncreaseMHP     uint32             `json:"increaseMHP"`
	IncreaseMMP     uint32             `json:"increaseMMP"`
	IncreasePAD     uint32             `json:"increasePAD"`
	IncreaseMAD     uint32             `json:"increaseMAD"`
	IncreaseEVA     uint32             `json:"increaseEVA"`
	IncreaseLUK     uint32             `json:"increaseLUK"`
	IncreaseDEX     uint32             `json:"increaseDEX"`
	IncreaseINT     uint32             `json:"increaseINT"`
	IncreaseSTR     uint32             `json:"increaseSTR"`
	IncreaseSpeed   uint32             `json:"increaseSpeed"`
	Spec            map[SpecType]int32 `json:"spec"`
	MonsterSummons  map[uint32]uint32  `json:"monsterSummons"`
	Morphs          map[uint32]uint32  `json:"morphs"`
	Skills          []uint32           `json:"skills"`
	Rewards         []RewardRestModel  `json:"rewards"`
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

type RewardRestModel struct {
	ItemId uint32 `json:"itemId"`
	Count  uint32 `json:"count"`
	Prob   uint32 `json:"prob"`
}
