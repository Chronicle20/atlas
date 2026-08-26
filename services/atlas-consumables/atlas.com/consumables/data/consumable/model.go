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

type SummonModel struct {
	templateId  uint32
	probability uint32
}

func (m SummonModel) Probability() uint32 {
	return m.probability
}

func (m SummonModel) TemplateId() uint32 {
	return m.templateId
}

type Model struct {
	id              uint32
	tradeBlock      bool
	price           uint32
	unitPrice       uint32
	slotMax         uint32
	timeLimited     bool
	notSale         bool
	reqLevel        uint32
	quest           bool
	only            bool
	consumeOnPickup bool
	success         uint32
	cursed          uint32
	create          uint32
	masterLevel     uint32
	reqSkillLevel   uint32
	tradeAvailable  int32
	noCancelMouse   bool
	pquest          bool
	left            int32
	right           int32
	top             int32
	bottom          int32
	bridleMsgType   uint32
	bridleProp      uint32
	bridlePropChg   float64
	useDelay        uint32
	delayMsg        string
	incFatigue      int32
	npc             uint32
	script          string
	runOnPickup     bool
	monsterBook     bool
	monsterId       uint32
	bigSize         bool
	tragetBlock     bool
	effect          string
	monsterHp       uint32
	worldMsg        string
	incPDD          uint32
	incMDD          uint32
	incACC          uint32
	incMHP          uint32
	incMMP          uint32
	incPAD          uint32
	incMAD          uint32
	incEVA          uint32
	incLUK          uint32
	incDEX          uint32
	incINT          uint32
	incSTR          uint32
	incSpeed        uint32
	incJump         uint32
	spec            map[SpecType]int32
	monsterSummons  []SummonModel
	morphs          map[uint32]uint32
	skills          []uint32
	rewards         []RewardModel
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) GetSpec(specType SpecType) (int32, bool) {
	val, ok := m.spec[specType]
	return val, ok
}

func (m Model) SuccessRate() uint32 {
	return m.success
}

func (m Model) MasterLevel() uint32 {
	return m.masterLevel
}

func (m Model) ReqSkillLevel() uint32 {
	return m.reqSkillLevel
}

func (m Model) Skills() []uint32 {
	return m.skills
}

func (m Model) StrengthIncrease() uint32 {
	return m.incSTR
}

func (m Model) DexterityIncrease() uint32 {
	return m.incDEX
}

func (m Model) IntelligenceIncrease() uint32 {
	return m.incINT
}

func (m Model) LuckIncrease() uint32 {
	return m.incLUK
}

func (m Model) MaxHPIncrease() uint32 {
	return m.incMHP
}

func (m Model) MaxMPIncrease() uint32 {
	return m.incMMP
}

func (m Model) WeaponAttackIncrease() uint32 {
	return m.incPAD
}

func (m Model) MagicAttackIncrease() uint32 {
	return m.incMAD
}

func (m Model) WeaponDefenseIncrease() uint32 {
	return m.incPDD
}

func (m Model) MagicDefenseIncrease() uint32 {
	return m.incMDD
}

func (m Model) AccuracyIncrease() uint32 {
	return m.incACC
}

func (m Model) AvoidabilityIncrease() uint32 {
	return m.incEVA
}

func (m Model) HandsIncrease() uint32 {
	return 0
}

func (m Model) SpeedIncrease() uint32 {
	return m.incSpeed
}

func (m Model) JumpIncrease() uint32 {
	return m.incJump
}

func (m Model) CursedRate() uint32 {
	return m.cursed
}

func (m Model) MonsterSummons() []SummonModel {
	return m.monsterSummons
}

// Morphs returns the item's morphRandom table (morph id -> weight). The
// returned map is the internal reference, matching the MonsterSummons()
// accessor convention; callers are read-only.
func (m Model) Morphs() map[uint32]uint32 {
	return m.morphs
}

func (m Model) Rewards() []RewardModel {
	return m.rewards
}

// Create is the item id a successful use produces — for a catch item, the
// reward granted in the caught monster's place.
func (m Model) Create() uint32 {
	return m.create
}

// MonsterId is the mob template a catch item targets (WZ info/mob).
func (m Model) MonsterId() uint32 {
	return m.monsterId
}

// MonsterHp is the catch HP gate (WZ info/mobHP), interpreted as a PERCENTAGE
// of the target's max HP (task-212 assumption A-1). Zero means no gate.
func (m Model) MonsterHp() uint32 {
	return m.monsterHp
}

// BridleMsgType selects the CLIENT-side "no monster found" message and is never
// read off the wire by either catch response packet (task-212 design §6.3).
func (m Model) BridleMsgType() uint32 {
	return m.bridleMsgType
}

// BridleProp is the base catch success percentage. Zero means deterministic
// once the species and HP gates pass.
func (m Model) BridleProp() uint32 {
	return m.bridleProp
}

// BridlePropChg multiplies BridleProp once, statelessly (assumption A-2).
func (m Model) BridlePropChg() float64 {
	return m.bridlePropChg
}

// UseDelay is the per-item cooldown in milliseconds (WZ info/useDelay).
func (m Model) UseDelay() uint32 {
	return m.useDelay
}

// DelayMsg is what BRIDLE_MOB_CATCH_FAIL reason 1 renders.
func (m Model) DelayMsg() string {
	return m.delayMsg
}

type RewardModel struct {
	itemId   uint32
	count    uint32
	prob     uint32
	effect   string
	worldMsg string
	period   int32
}

func (m RewardModel) ItemId() uint32 {
	return m.itemId
}

func (m RewardModel) Count() uint32 {
	return m.count
}

func (m RewardModel) Prob() uint32 {
	return m.prob
}

func (m RewardModel) Effect() string {
	return m.effect
}

func (m RewardModel) WorldMsg() string {
	return m.worldMsg
}

func (m RewardModel) Period() int32 {
	return m.period
}

type RewardModelBuilderType struct {
	m RewardModel
}

func RewardModelBuilder() *RewardModelBuilderType { return &RewardModelBuilderType{} }

func (b *RewardModelBuilderType) SetItemId(v uint32) *RewardModelBuilderType {
	b.m.itemId = v
	return b
}

func (b *RewardModelBuilderType) SetCount(v uint32) *RewardModelBuilderType { b.m.count = v; return b }

func (b *RewardModelBuilderType) SetProb(v uint32) *RewardModelBuilderType { b.m.prob = v; return b }

func (b *RewardModelBuilderType) SetEffect(v string) *RewardModelBuilderType {
	b.m.effect = v
	return b
}

func (b *RewardModelBuilderType) SetWorldMsg(v string) *RewardModelBuilderType {
	b.m.worldMsg = v
	return b
}

func (b *RewardModelBuilderType) SetPeriod(v int32) *RewardModelBuilderType { b.m.period = v; return b }
func (b *RewardModelBuilderType) Build() RewardModel                        { return b.m }
