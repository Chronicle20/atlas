package effect

import (
	"atlas-data/skill/effect/statup"
)

type RestModel struct {
	WeaponAttack      int16   `json:"weaponAttack"`
	MagicAttack       int16   `json:"magicAttack"`
	WeaponDefense     int16   `json:"weaponDefense"`
	MagicDefense      int16   `json:"magicDefense"`
	Accuracy          int16   `json:"accuracy"`
	Avoidability      int16   `json:"avoidability"`
	Speed             int16   `json:"speed"`
	Jump              int16   `json:"jump"`
	Hp                uint16  `json:"hp"`
	Mp                uint16  `json:"mp"`
	HPR               float64 `json:"hpR"`
	MPR               float64 `json:"mpR"`
	MHPRRate          uint16  `json:"MHPRRate"`
	MMPRRate          uint16  `json:"MMPRRate"`
	MobSkill          uint16  `json:"mobSkill"`
	MobSkillLevel     uint16  `json:"mobSkillLevel"`
	MHPR              byte    `json:"mhpr"`
	MMPR              byte    `json:"mmpr"`
	HPConsume         uint16  `json:"HPConsume"`
	MPConsume         uint16  `json:"MPConsume"`
	Duration          int32   `json:"duration"`
	Target            uint32  `json:"target"`
	Barrier           int32   `json:"barrier"`
	Mob               uint32  `json:"mob"`
	OverTime          bool    `json:"overTime"`
	RepeatEffect      bool    `json:"repeatEffect"`
	MoveTo            int32   `json:"moveTo"`
	CP                uint32  `json:"cp"`
	NuffSkill         uint32  `json:"nuffSkill"`
	Skill             bool    `json:"skill"`
	X                 int16   `json:"x"`
	Y                 int16   `json:"y"`
	MobCount          uint32  `json:"mobCount"`
	MoneyConsume      uint32  `json:"moneyConsume"`
	Cooldown          uint32  `json:"cooldown"`
	MorphId           uint32  `json:"morphId"`
	Ghost             uint32  `json:"ghost"`
	Fatigue           uint32  `json:"fatigue"`
	Berserk           uint32  `json:"berserk"`
	Booster           uint32  `json:"booster"`
	Prop              float64 `json:"prop"`
	ItemConsume       uint32  `json:"itemConsume"`
	ItemConsumeAmount uint32  `json:"itemConsumeAmount"`
	Damage            uint32  `json:"damage"`
	AttackCount       uint32  `json:"attackCount"`
	FixDamage         int32   `json:"fixDamage"`
	// Dot is the raw per-tick damage-over-time magnitude (WZ `dot`).
	// Forwarded unscaled.
	Dot int32 `json:"dot"`
	// DotInterval is the DoT tick interval in MILLISECONDS. WZ stores
	// seconds; the reader converts (task-054 unit contract).
	DotInterval int32 `json:"dotInterval"`
	// DotTime is the DoT lifetime in MILLISECONDS. WZ stores seconds; the
	// reader converts.
	DotTime              int32              `json:"dotTime"`
	LT                   *PointRestModel    `json:"lt,omitempty"`
	RB                   *PointRestModel    `json:"rb,omitempty"`
	BulletCount          uint16             `json:"bulletCount"`
	BulletConsume        uint16             `json:"bulletConsume"`
	MapProtection        byte               `json:"mapProtection"`
	CureAbnormalStatuses []string           `json:"cureAbnormalStatuses"`
	Statups              []statup.RestModel `json:"statups"`
	MonsterStatus        map[string]uint32  `json:"monsterStatus"`
	CardStats            cardItemUp         `json:"cardStats"`

	// Fields below are Skill.wz `common` keys (task-192). Go name is the
	// PascalCase of the wz key and the JSON tag is the wz key verbatim; the
	// semantics of most of them are unverified, so no key is given an
	// invented descriptive name. Populated from both the `common` and
	// `level` read paths.
	Range             int32 `json:"range"`
	Mastery           int32 `json:"mastery"`
	Z                 int32 `json:"z"`
	Cr                int32 `json:"cr"`
	DamR              int32 `json:"damR"`
	CriticaldamageMin int32 `json:"criticaldamageMin"`
	V                 int32 `json:"v"`
	IgnoreMobpdpR     int32 `json:"ignoreMobpdpR"`
	Epad              int32 `json:"epad"`
	W                 int32 `json:"w"`
	U                 int32 `json:"u"`
	Epdd              int32 `json:"epdd"`
	Emdd              int32 `json:"emdd"`
	SelfDestruction   int32 `json:"selfDestruction"`
	AsrR              int32 `json:"asrR"`
	T                 int32 `json:"t"`
	Er                int32 `json:"er"`
	PddR              int32 `json:"pddR"`
	TerR              int32 `json:"terR"`
	MadX              int32 `json:"madX"`
	SubProp           int32 `json:"subProp"`
	Emhp              int32 `json:"emhp"`
	CriticaldamageMax int32 `json:"criticaldamageMax"`
	ExpR              int32 `json:"expR"`
	Emmp              int32 `json:"emmp"`
	// ConsumeItemId is wz `common/itemConsume`. It is deliberately NOT the
	// `itemConsume` JSON attribute above, which is wz `itemCon` — the two are
	// distinct keys that never co-occur, and folding them would silently
	// merge two differently-sourced values (FR-6.4, design §5.4).
	ConsumeItemId int32 `json:"consumeItemId"`
	MddR          int32 `json:"mddR"`
	SubTime       int32 `json:"subTime"`
	PadX          int32 `json:"padX"`
	MesoR         int32 `json:"mesoR"`
}

type cardItemUp struct {
	ItemCode    uint32 `json:"itemCode"`
	Probability uint32 `json:"probability"`
	Areas       []area `json:"areas"`
	InParty     bool   `json:"inParty"`
}

type area struct {
	Start uint32 `json:"start"`
	End   uint32 `json:"end"`
}

// Transform converts the immutable domain Model into its wire representation.
// LT/RB collapse to nil when the rectangle is the zero point, matching the
// wz `time`-absent-vector convention the client expects (an absent rectangle
// serializes as JSON null, not {"x":0,"y":0}). CardStats has no builder-side
// setter (nothing populates it yet), so it always transforms to its zero
// value — unchanged from the prior direct-to-RestModel behavior.
func Transform(m Model) (RestModel, error) {
	var ltPtr *PointRestModel
	if m.LT().X() != 0 || m.LT().Y() != 0 {
		ltPtr = &PointRestModel{X: int16(m.LT().X()), Y: int16(m.LT().Y())}
	}
	var rbPtr *PointRestModel
	if m.RB().X() != 0 || m.RB().Y() != 0 {
		rbPtr = &PointRestModel{X: int16(m.RB().X()), Y: int16(m.RB().Y())}
	}
	return RestModel{
		WeaponAttack:         m.WeaponAttack(),
		MagicAttack:          m.MagicAttack(),
		WeaponDefense:        m.WeaponDefense(),
		MagicDefense:         m.MagicDefense(),
		Accuracy:             m.Accuracy(),
		Avoidability:         m.Avoidability(),
		Speed:                m.Speed(),
		Jump:                 m.Jump(),
		Hp:                   m.Hp(),
		Mp:                   m.Mp(),
		HPR:                  m.HPR(),
		MPR:                  m.MPR(),
		MHPRRate:             m.MHPRRate(),
		MMPRRate:             m.MMPRRate(),
		MobSkill:             m.MobSkill(),
		MobSkillLevel:        m.MobSkillLevel(),
		MHPR:                 m.MHPR(),
		MMPR:                 m.MMPR(),
		HPConsume:            m.HPConsume(),
		MPConsume:            m.MPConsume(),
		Duration:             m.Duration(),
		Target:               m.Target(),
		Barrier:              m.Barrier(),
		Mob:                  m.Mob(),
		OverTime:             m.OverTime(),
		RepeatEffect:         m.RepeatEffect(),
		MoveTo:               m.MoveTo(),
		CP:                   m.CP(),
		NuffSkill:            m.NuffSkill(),
		Skill:                m.Skill(),
		X:                    m.X(),
		Y:                    m.Y(),
		MobCount:             m.MobCount(),
		MoneyConsume:         m.MoneyConsume(),
		Cooldown:             m.Cooldown(),
		MorphId:              m.MorphId(),
		Ghost:                m.Ghost(),
		Fatigue:              m.Fatigue(),
		Berserk:              m.Berserk(),
		Booster:              m.Booster(),
		Prop:                 m.Prop(),
		ItemConsume:          m.ItemConsume(),
		ItemConsumeAmount:    m.ItemConsumeAmount(),
		Damage:               m.Damage(),
		AttackCount:          m.AttackCount(),
		FixDamage:            m.FixDamage(),
		LT:                   ltPtr,
		RB:                   rbPtr,
		BulletCount:          m.BulletCount(),
		BulletConsume:        m.BulletConsume(),
		MapProtection:        m.MapProtection(),
		CureAbnormalStatuses: m.CureAbnormalStatuses(),
		Statups:              m.Statups(),
		MonsterStatus:        m.MonsterStatus(),
		Range:                m.Range(),
		Mastery:              m.Mastery(),
		Z:                    m.Z(),
		Dot:                  m.Dot(),
		Cr:                   m.Cr(),
		DotInterval:          m.DotInterval(),
		DotTime:              m.DotTime(),
		DamR:                 m.DamR(),
		CriticaldamageMin:    m.CriticaldamageMin(),
		V:                    m.V(),
		IgnoreMobpdpR:        m.IgnoreMobpdpR(),
		Epad:                 m.Epad(),
		W:                    m.W(),
		U:                    m.U(),
		Epdd:                 m.Epdd(),
		Emdd:                 m.Emdd(),
		SelfDestruction:      m.SelfDestruction(),
		AsrR:                 m.AsrR(),
		T:                    m.T(),
		Er:                   m.Er(),
		PddR:                 m.PddR(),
		TerR:                 m.TerR(),
		MadX:                 m.MadX(),
		SubProp:              m.SubProp(),
		Emhp:                 m.Emhp(),
		CriticaldamageMax:    m.CriticaldamageMax(),
		ExpR:                 m.ExpR(),
		Emmp:                 m.Emmp(),
		ConsumeItemId:        m.ConsumeItemId(),
		MddR:                 m.MddR(),
		SubTime:              m.SubTime(),
		PadX:                 m.PadX(),
		MesoR:                m.MesoR(),
	}, nil
}

// TransformAll converts a slice of domain Models to their wire representation.
func TransformAll(ms []Model) ([]RestModel, error) {
	rms := make([]RestModel, 0, len(ms))
	for _, m := range ms {
		rm, err := Transform(m)
		if err != nil {
			return nil, err
		}
		rms = append(rms, rm)
	}
	return rms, nil
}
