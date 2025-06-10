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
	tradeAvailable  bool
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
	monsterSummons  map[uint32]uint32
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

func (m Model) MonsterSummons() map[uint32]uint32 {
	return m.monsterSummons
}

type RewardModel struct {
	itemId uint32
	count  uint32
	prob   uint32
}
