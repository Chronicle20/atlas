package consumable

import (
	"github.com/Chronicle20/atlas-model/model"
	"strconv"
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
	TragetBlock     bool               `json:"tragetBlock"` // Assuming typo for "TargetBlock"
	Effect          string             `json:"effect"`
	MonsterHP       uint32             `json:"monsterHP"`
	WorldMsg        string             `json:"worldMsg"`
	Increase        uint32             `json:"increase"`
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

func Transform(m Model) (RestModel, error) {
	rs, err := model.SliceMap(TransformReward)(model.FixedProvider(m.rewards))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		Id:              m.id,
		TradeBlock:      m.tradeBlock,
		Price:           m.price,
		UnitPrice:       m.unitPrice,
		SlotMax:         m.slotMax,
		TimeLimited:     m.timeLimited,
		NotSale:         m.notSale,
		ReqLevel:        m.reqLevel,
		Quest:           m.quest,
		Only:            m.only,
		ConsumeOnPickup: m.consumeOnPickup,
		Success:         m.success,
		Cursed:          m.cursed,
		Create:          m.create,
		MasterLevel:     m.masterLevel,
		ReqSkillLevel:   m.reqSkillLevel,
		TradeAvailable:  m.tradeAvailable,
		NoCancelMouse:   m.noCancelMouse,
		Pquest:          m.pquest,
		Left:            m.left,
		Right:           m.right,
		Top:             m.top,
		Bottom:          m.bottom,
		BridleMsgType:   m.bridleMsgType,
		BridleProp:      m.bridleProp,
		BridlePropChg:   m.bridlePropChg,
		UseDelay:        m.useDelay,
		DelayMsg:        m.delayMsg,
		IncFatigue:      m.incFatigue,
		Npc:             m.npc,
		Script:          m.script,
		RunOnPickup:     m.runOnPickup,
		MonsterBook:     m.monsterBook,
		MonsterId:       m.monsterId,
		BigSize:         m.bigSize,
		TragetBlock:     m.tragetBlock,
		Effect:          m.effect,
		MonsterHP:       m.monsterHp,
		WorldMsg:        m.worldMsg,
		Increase:        m.inc,
		IncreasePDD:     m.incPDD,
		IncreaseMDD:     m.incMDD,
		IncreaseACC:     m.incACC,
		IncreaseMHP:     m.incMHP,
		IncreaseMMP:     m.incMMP,
		IncreasePAD:     m.incPAD,
		IncreaseMAD:     m.incMAD,
		IncreaseEVA:     m.incEVA,
		IncreaseLUK:     m.incLUK,
		IncreaseDEX:     m.incDEX,
		IncreaseINT:     m.incINT,
		IncreaseSTR:     m.incSTR,
		IncreaseSpeed:   m.incSpeed,
		Spec:            m.spec,
		MonsterSummons:  m.monsterSummons,
		Morphs:          m.morphs,
		Skills:          m.skills,
		Rewards:         rs,
	}, nil
}

type RewardRestModel struct {
	ItemId uint32 `json:"itemId"`
	Count  uint32 `json:"count"`
	Prob   uint32 `json:"prob"`
}

func TransformReward(m RewardModel) (RewardRestModel, error) {
	return RewardRestModel{
		ItemId: m.itemId,
		Count:  m.count,
		Prob:   m.prob,
	}, nil
}
