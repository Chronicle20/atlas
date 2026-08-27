package consumable

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type Summon struct {
	TemplateId  uint32 `json:"templateId"`
	Probability uint32 `json:"probability"`
}

type RestModel struct {
	Id              uint32             `json:"-"`
	TradeBlock      bool               `json:"tradeBlock"`
	Price           uint32             `json:"price"`
	UnitPrice       float64            `json:"unitPrice"`
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
	TradeAvailable  int32              `json:"tradeAvailable"`
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
	IncreaseJump    uint32             `json:"increaseJump"`
	IncreaseEVA     uint32             `json:"increaseEVA"`
	IncreaseLUK     uint32             `json:"increaseLUK"`
	IncreaseDEX     uint32             `json:"increaseDEX"`
	IncreaseINT     uint32             `json:"increaseINT"`
	IncreaseSTR     uint32             `json:"increaseSTR"`
	IncreaseSpeed   uint32             `json:"increaseSpeed"`
	Spec            map[SpecType]int32 `json:"spec"`
	MonsterSummons  []Summon           `json:"monsterSummons"`
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
	rewards := make([]RewardRestModel, 0, len(m.rewards))
	for _, r := range m.rewards {
		rr, err := TransformReward(r)
		if err != nil {
			return RestModel{}, err
		}
		rewards = append(rewards, rr)
	}

	summons := make([]Summon, 0, len(m.monsterSummons))
	for _, s := range m.monsterSummons {
		summons = append(summons, Summon{
			TemplateId:  s.templateId,
			Probability: s.probability,
		})
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
		TargetBlock:     m.tragetBlock,
		Effect:          m.effect,
		MonsterHP:       m.monsterHp,
		WorldMsg:        m.worldMsg,
		IncreasePDD:     m.incPDD,
		IncreaseMDD:     m.incMDD,
		IncreaseACC:     m.incACC,
		IncreaseMHP:     m.incMHP,
		IncreaseMMP:     m.incMMP,
		IncreasePAD:     m.incPAD,
		IncreaseMAD:     m.incMAD,
		IncreaseJump:    m.incJump,
		IncreaseEVA:     m.incEVA,
		IncreaseLUK:     m.incLUK,
		IncreaseDEX:     m.incDEX,
		IncreaseINT:     m.incINT,
		IncreaseSTR:     m.incSTR,
		IncreaseSpeed:   m.incSpeed,
		Spec:            m.spec,
		MonsterSummons:  summons,
		Morphs:          m.morphs,
		Skills:          m.skills,
		Rewards:         rewards,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	rs, err := model.SliceMap(ExtractReward)(model.FixedProvider(rm.Rewards))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}
	ms, err := model.SliceMap(func(m Summon) (SummonModel, error) {
		return SummonModel{
			templateId:  m.TemplateId,
			probability: m.Probability,
		}, nil
	})(model.FixedProvider(rm.MonsterSummons))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}
	return Model{
		id:              rm.Id,
		tradeBlock:      rm.TradeBlock,
		price:           rm.Price,
		unitPrice:       rm.UnitPrice,
		slotMax:         rm.SlotMax,
		timeLimited:     rm.TimeLimited,
		notSale:         rm.NotSale,
		reqLevel:        rm.ReqLevel,
		quest:           rm.Quest,
		only:            rm.Only,
		consumeOnPickup: rm.ConsumeOnPickup,
		success:         rm.Success,
		cursed:          rm.Cursed,
		create:          rm.Create,
		masterLevel:     rm.MasterLevel,
		reqSkillLevel:   rm.ReqSkillLevel,
		tradeAvailable:  rm.TradeAvailable,
		noCancelMouse:   rm.NoCancelMouse,
		pquest:          rm.Pquest,
		left:            rm.Left,
		right:           rm.Right,
		top:             rm.Top,
		bottom:          rm.Bottom,
		bridleMsgType:   rm.BridleMsgType,
		bridleProp:      rm.BridleProp,
		bridlePropChg:   rm.BridlePropChg,
		useDelay:        rm.UseDelay,
		delayMsg:        rm.DelayMsg,
		incFatigue:      rm.IncFatigue,
		npc:             rm.Npc,
		script:          rm.Script,
		runOnPickup:     rm.RunOnPickup,
		monsterBook:     rm.MonsterBook,
		monsterId:       rm.MonsterId,
		bigSize:         rm.BigSize,
		tragetBlock:     rm.TargetBlock,
		effect:          rm.Effect,
		monsterHp:       rm.MonsterHP,
		worldMsg:        rm.WorldMsg,
		incPDD:          rm.IncreasePDD,
		incMDD:          rm.IncreaseMDD,
		incACC:          rm.IncreaseACC,
		incMHP:          rm.IncreaseMHP,
		incMMP:          rm.IncreaseMMP,
		incPAD:          rm.IncreasePAD,
		incMAD:          rm.IncreaseMAD,
		incEVA:          rm.IncreaseEVA,
		incLUK:          rm.IncreaseLUK,
		incDEX:          rm.IncreaseDEX,
		incINT:          rm.IncreaseINT,
		incSTR:          rm.IncreaseSTR,
		incSpeed:        rm.IncreaseSpeed,
		incJump:         rm.IncreaseJump,
		spec:            rm.Spec,
		monsterSummons:  ms,
		morphs:          rm.Morphs,
		skills:          rm.Skills,
		rewards:         rs,
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

func ExtractReward(rm RewardRestModel) (RewardModel, error) {
	return RewardModel{
		itemId: rm.ItemId,
		count:  rm.Count,
		prob:   rm.Prob,
	}, nil
}
