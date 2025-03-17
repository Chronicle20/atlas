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

func Extract(rm RestModel) (Model, error) {
	rs, err := model.SliceMap(ExtractReward)(model.FixedProvider(rm.Rewards))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}
	return Model{
		Id:              rm.Id,
		TradeBlock:      rm.TradeBlock,
		Price:           rm.Price,
		UnitPrice:       rm.UnitPrice,
		SlotMax:         rm.SlotMax,
		TimeLimited:     rm.TimeLimited,
		NotSale:         rm.NotSale,
		ReqLevel:        rm.ReqLevel,
		Quest:           rm.Quest,
		Only:            rm.Only,
		ConsumeOnPickup: rm.ConsumeOnPickup,
		Success:         rm.Success,
		Cursed:          rm.Cursed,
		Create:          rm.Create,
		MasterLevel:     rm.MasterLevel,
		ReqSkillLevel:   rm.ReqSkillLevel,
		TradeAvailable:  rm.TradeAvailable,
		NoCancelMouse:   rm.NoCancelMouse,
		Pquest:          rm.Pquest,
		Left:            rm.Left,
		Right:           rm.Right,
		Top:             rm.Top,
		Bottom:          rm.Bottom,
		BridleMsgType:   rm.BridleMsgType,
		BridleProp:      rm.BridleProp,
		BridlePropChg:   rm.BridlePropChg,
		UseDelay:        rm.UseDelay,
		DelayMsg:        rm.DelayMsg,
		IncFatigue:      rm.IncFatigue,
		Npc:             rm.Npc,
		Script:          rm.Script,
		RunOnPickup:     rm.RunOnPickup,
		MonsterBook:     rm.MonsterBook,
		MonsterId:       rm.MonsterId,
		BigSize:         rm.BigSize,
		TargetBlock:     rm.TragetBlock,
		Effect:          rm.Effect,
		MonsterHp:       rm.MonsterHP,
		WorldMsg:        rm.WorldMsg,
		IncPDD:          rm.IncreasePDD,
		IncMDD:          rm.IncreaseMDD,
		IncACC:          rm.IncreaseACC,
		IncMHP:          rm.IncreaseMHP,
		IncMMP:          rm.IncreaseMMP,
		IncPAD:          rm.IncreasePAD,
		IncMAD:          rm.IncreaseMAD,
		IncEVA:          rm.IncreaseEVA,
		IncLUK:          rm.IncreaseLUK,
		IncDEX:          rm.IncreaseDEX,
		IncINT:          rm.IncreaseINT,
		IncSTR:          rm.IncreaseSTR,
		IncSpeed:        rm.IncreaseSpeed,
		Spec:            rm.Spec,
		MonsterSummons:  rm.MonsterSummons,
		Morphs:          rm.Morphs,
		Skills:          rm.Skills,
		Rewards:         rs,
	}, nil
}

func Transform(m Model) (RestModel, error) {
	rs, err := model.SliceMap(TransformReward)(model.FixedProvider(m.Rewards))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		Id:              m.Id,
		TradeBlock:      m.TradeBlock,
		Price:           m.Price,
		UnitPrice:       m.UnitPrice,
		SlotMax:         m.SlotMax,
		TimeLimited:     m.TimeLimited,
		NotSale:         m.NotSale,
		ReqLevel:        m.ReqLevel,
		Quest:           m.Quest,
		Only:            m.Only,
		ConsumeOnPickup: m.ConsumeOnPickup,
		Success:         m.Success,
		Cursed:          m.Cursed,
		Create:          m.Create,
		MasterLevel:     m.MasterLevel,
		ReqSkillLevel:   m.ReqSkillLevel,
		TradeAvailable:  m.TradeAvailable,
		NoCancelMouse:   m.NoCancelMouse,
		Pquest:          m.Pquest,
		Left:            m.Left,
		Right:           m.Right,
		Top:             m.Top,
		Bottom:          m.Bottom,
		BridleMsgType:   m.BridleMsgType,
		BridleProp:      m.BridleProp,
		BridlePropChg:   m.BridlePropChg,
		UseDelay:        m.UseDelay,
		DelayMsg:        m.DelayMsg,
		IncFatigue:      m.IncFatigue,
		Npc:             m.Npc,
		Script:          m.Script,
		RunOnPickup:     m.RunOnPickup,
		MonsterBook:     m.MonsterBook,
		MonsterId:       m.MonsterId,
		BigSize:         m.BigSize,
		TragetBlock:     m.TargetBlock,
		Effect:          m.Effect,
		MonsterHP:       m.MonsterHp,
		WorldMsg:        m.WorldMsg,
		IncreasePDD:     m.IncPDD,
		IncreaseMDD:     m.IncMDD,
		IncreaseACC:     m.IncACC,
		IncreaseMHP:     m.IncMHP,
		IncreaseMMP:     m.IncMMP,
		IncreasePAD:     m.IncPAD,
		IncreaseMAD:     m.IncMAD,
		IncreaseEVA:     m.IncEVA,
		IncreaseLUK:     m.IncLUK,
		IncreaseDEX:     m.IncDEX,
		IncreaseINT:     m.IncINT,
		IncreaseSTR:     m.IncSTR,
		IncreaseSpeed:   m.IncSpeed,
		Spec:            m.Spec,
		MonsterSummons:  m.MonsterSummons,
		Morphs:          m.Morphs,
		Skills:          m.Skills,
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
		ItemId: m.ItemId,
		Count:  m.Count,
		Prob:   m.Prob,
	}, nil
}

func ExtractReward(rm RewardRestModel) (RewardModel, error) {
	return RewardModel{
		ItemId: rm.ItemId,
		Count:  rm.Count,
		Prob:   rm.Prob,
	}, nil
}
