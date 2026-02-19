package consumable

import "encoding/json"

type jsonModel struct {
	Id              uint32             `json:"id"`
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
	IncreaseJump    uint32             `json:"increaseJump"`
	Spec            map[SpecType]int32 `json:"spec"`
	MonsterSummons  []jsonSummon       `json:"monsterSummons"`
	Morphs          map[uint32]uint32  `json:"morphs"`
	Skills          []uint32           `json:"skills"`
	Rewards         []jsonReward       `json:"rewards"`
}

type jsonSummon struct {
	TemplateId  uint32 `json:"templateId"`
	Probability uint32 `json:"probability"`
}

type jsonReward struct {
	ItemId uint32 `json:"itemId"`
	Count  uint32 `json:"count"`
	Prob   uint32 `json:"prob"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	summons := make([]jsonSummon, len(m.monsterSummons))
	for i, s := range m.monsterSummons {
		summons[i] = jsonSummon{TemplateId: s.templateId, Probability: s.probability}
	}
	rewards := make([]jsonReward, len(m.rewards))
	for i, r := range m.rewards {
		rewards[i] = jsonReward{ItemId: r.itemId, Count: r.count, Prob: r.prob}
	}
	return json.Marshal(jsonModel{
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
		IncreaseEVA:     m.incEVA,
		IncreaseLUK:     m.incLUK,
		IncreaseDEX:     m.incDEX,
		IncreaseINT:     m.incINT,
		IncreaseSTR:     m.incSTR,
		IncreaseSpeed:   m.incSpeed,
		IncreaseJump:    m.incJump,
		Spec:            m.spec,
		MonsterSummons:  summons,
		Morphs:          m.morphs,
		Skills:          m.skills,
		Rewards:         rewards,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var jm jsonModel
	if err := json.Unmarshal(data, &jm); err != nil {
		return err
	}
	summons := make([]SummonModel, len(jm.MonsterSummons))
	for i, s := range jm.MonsterSummons {
		summons[i] = SummonModel{templateId: s.TemplateId, probability: s.Probability}
	}
	rewards := make([]RewardModel, len(jm.Rewards))
	for i, r := range jm.Rewards {
		rewards[i] = RewardModel{itemId: r.ItemId, count: r.Count, prob: r.Prob}
	}
	*m = Model{
		id:              jm.Id,
		tradeBlock:      jm.TradeBlock,
		price:           jm.Price,
		unitPrice:       jm.UnitPrice,
		slotMax:         jm.SlotMax,
		timeLimited:     jm.TimeLimited,
		notSale:         jm.NotSale,
		reqLevel:        jm.ReqLevel,
		quest:           jm.Quest,
		only:            jm.Only,
		consumeOnPickup: jm.ConsumeOnPickup,
		success:         jm.Success,
		cursed:          jm.Cursed,
		create:          jm.Create,
		masterLevel:     jm.MasterLevel,
		reqSkillLevel:   jm.ReqSkillLevel,
		tradeAvailable:  jm.TradeAvailable,
		noCancelMouse:   jm.NoCancelMouse,
		pquest:          jm.Pquest,
		left:            jm.Left,
		right:           jm.Right,
		top:             jm.Top,
		bottom:          jm.Bottom,
		bridleMsgType:   jm.BridleMsgType,
		bridleProp:      jm.BridleProp,
		bridlePropChg:   jm.BridlePropChg,
		useDelay:        jm.UseDelay,
		delayMsg:        jm.DelayMsg,
		incFatigue:      jm.IncFatigue,
		npc:             jm.Npc,
		script:          jm.Script,
		runOnPickup:     jm.RunOnPickup,
		monsterBook:     jm.MonsterBook,
		monsterId:       jm.MonsterId,
		bigSize:         jm.BigSize,
		tragetBlock:     jm.TargetBlock,
		effect:          jm.Effect,
		monsterHp:       jm.MonsterHP,
		worldMsg:        jm.WorldMsg,
		incPDD:          jm.IncreasePDD,
		incMDD:          jm.IncreaseMDD,
		incACC:          jm.IncreaseACC,
		incMHP:          jm.IncreaseMHP,
		incMMP:          jm.IncreaseMMP,
		incPAD:          jm.IncreasePAD,
		incMAD:          jm.IncreaseMAD,
		incEVA:          jm.IncreaseEVA,
		incLUK:          jm.IncreaseLUK,
		incDEX:          jm.IncreaseDEX,
		incINT:          jm.IncreaseINT,
		incSTR:          jm.IncreaseSTR,
		incSpeed:        jm.IncreaseSpeed,
		incJump:         jm.IncreaseJump,
		spec:            jm.Spec,
		monsterSummons:  summons,
		morphs:          jm.Morphs,
		skills:          jm.Skills,
		rewards:         rewards,
	}
	return nil
}
