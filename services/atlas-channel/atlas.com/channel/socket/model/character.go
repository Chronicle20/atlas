package model

import (
	"atlas-channel/tool"
	"errors"
	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"sort"
	"time"
)

type CharacterTemporaryStatType struct {
	shift   uint
	mask    tool.Uint128
	disease bool
}

func (t CharacterTemporaryStatType) Shift() uint {
	return t.shift
}

func NewCharacterTemporaryStatType(shift uint, disease bool) CharacterTemporaryStatType {
	mask := tool.Uint128{L: 1}.ShiftLeft(shift)
	return CharacterTemporaryStatType{
		shift:   shift,
		mask:    mask,
		disease: disease,
	}
}

func CharacterTemporaryStatTypeByName(tenant tenant.Model) func(name character.TemporaryStatType) (CharacterTemporaryStatType, error) {
	var shift uint = 0
	set := make(map[character.TemporaryStatType]CharacterTemporaryStatType)

	funcCallNewAndInc := func(disease bool) func(name character.TemporaryStatType) {
		return func(name character.TemporaryStatType) {
			set[name] = NewCharacterTemporaryStatType(shift, disease)
			shift += 1
		}
	}
	newAndIncDiseased := funcCallNewAndInc(true)
	newAndIncNonDiseased := funcCallNewAndInc(false)

	newAndIncNonDiseased(character.TemporaryStatTypeWeaponAttack)
	newAndIncNonDiseased(character.TemporaryStatTypeWeaponDefense)
	newAndIncNonDiseased(character.TemporaryStatTypeMagicAttack)
	newAndIncNonDiseased(character.TemporaryStatTypeMagicDefense)
	newAndIncNonDiseased(character.TemporaryStatTypeAccuracy)
	newAndIncNonDiseased(character.TemporaryStatTypeAvoidability)
	newAndIncNonDiseased(character.TemporaryStatTypeHands)
	newAndIncNonDiseased(character.TemporaryStatTypeSpeed)
	newAndIncNonDiseased(character.TemporaryStatTypeJump)
	newAndIncNonDiseased(character.TemporaryStatTypeMagicGuard)
	newAndIncNonDiseased(character.TemporaryStatTypeDarkSight)
	newAndIncNonDiseased(character.TemporaryStatTypeBooster)
	newAndIncNonDiseased(character.TemporaryStatTypePowerGuard)
	newAndIncNonDiseased(character.TemporaryStatTypeHyperBodyHP)
	newAndIncNonDiseased(character.TemporaryStatTypeHyperBodyMP)
	newAndIncNonDiseased(character.TemporaryStatTypeInvincible)
	newAndIncNonDiseased(character.TemporaryStatTypeSoulArrow)
	newAndIncDiseased(character.TemporaryStatTypeStun)
	newAndIncDiseased(character.TemporaryStatTypePoison)
	newAndIncDiseased(character.TemporaryStatTypeSeal)
	newAndIncDiseased(character.TemporaryStatTypeDarkness)
	newAndIncNonDiseased(character.TemporaryStatTypeCombo)
	newAndIncNonDiseased(character.TemporaryStatTypeWhiteKnightCharge)
	newAndIncNonDiseased(character.TemporaryStatTypeDragonBlood)
	newAndIncNonDiseased(character.TemporaryStatTypeHolySymbol)
	newAndIncNonDiseased(character.TemporaryStatTypeMesoUp)
	newAndIncNonDiseased(character.TemporaryStatTypeShadowPartner)
	newAndIncNonDiseased(character.TemporaryStatTypePickPocket)
	newAndIncNonDiseased(character.TemporaryStatTypeMesoGuard)
	newAndIncNonDiseased(character.TemporaryStatTypeThaw)
	newAndIncDiseased(character.TemporaryStatTypeWeaken)
	newAndIncDiseased(character.TemporaryStatTypeCurse)
	newAndIncNonDiseased(character.TemporaryStatTypeSlow)
	newAndIncNonDiseased(character.TemporaryStatTypeMorph)
	newAndIncNonDiseased(character.TemporaryStatTypeRecovery)
	newAndIncNonDiseased(character.TemporaryStatTypeMapleWarrior)
	newAndIncNonDiseased(character.TemporaryStatTypeStance)
	newAndIncNonDiseased(character.TemporaryStatTypeSharpEyes)
	newAndIncNonDiseased(character.TemporaryStatTypeManaReflection)
	newAndIncDiseased(character.TemporaryStatTypeSeduce)
	newAndIncNonDiseased(character.TemporaryStatTypeShadowClaw)
	newAndIncNonDiseased(character.TemporaryStatTypeInfinity)
	newAndIncNonDiseased(character.TemporaryStatTypeHolyShield)
	newAndIncNonDiseased(character.TemporaryStatTypeHamstring)
	newAndIncNonDiseased(character.TemporaryStatTypeBlind)
	newAndIncNonDiseased(character.TemporaryStatTypeConcentrate)
	newAndIncNonDiseased(character.TemporaryStatTypeBanMap)
	newAndIncNonDiseased(character.TemporaryStatTypeEchoOfHero)
	newAndIncNonDiseased(character.TemporaryStatTypeMesoUpByItem)
	newAndIncNonDiseased(character.TemporaryStatTypeGhostMorph)
	newAndIncNonDiseased(character.TemporaryStatTypeBarrier)
	newAndIncDiseased(character.TemporaryStatTypeConfuse)
	newAndIncNonDiseased(character.TemporaryStatTypeItemUpByItem)
	newAndIncNonDiseased(character.TemporaryStatTypeRespectPImmune)
	newAndIncNonDiseased(character.TemporaryStatTypeRespectMImmune)
	newAndIncNonDiseased(character.TemporaryStatTypeDefenseAttack)
	newAndIncNonDiseased(character.TemporaryStatTypeDefenseState)
	newAndIncNonDiseased(character.TemporaryStatTypeIncreaseEffectHpPotion)
	newAndIncNonDiseased(character.TemporaryStatTypeIncreaseEffectMpPotion)
	newAndIncNonDiseased(character.TemporaryStatTypeBerserkFury)
	newAndIncNonDiseased(character.TemporaryStatTypeDivineBody)
	newAndIncNonDiseased(character.TemporaryStatTypeSpark)
	newAndIncNonDiseased(character.TemporaryStatTypeDojangShield)
	newAndIncNonDiseased(character.TemporaryStatTypeSoulMasterFinal)
	newAndIncNonDiseased(character.TemporaryStatTypeWindBreakerFinal)
	newAndIncNonDiseased(character.TemporaryStatTypeElementalReset)
	newAndIncNonDiseased(character.TemporaryStatTypeWindWalk)
	newAndIncNonDiseased(character.TemporaryStatTypeEventRate)
	newAndIncNonDiseased(character.TemporaryStatTypeAranCombo)
	newAndIncNonDiseased(character.TemporaryStatTypeComboDrain)
	newAndIncNonDiseased(character.TemporaryStatTypeComboBarrier)
	newAndIncNonDiseased(character.TemporaryStatTypeBodyPressure)
	newAndIncNonDiseased(character.TemporaryStatTypeSmartKnockBack)
	newAndIncNonDiseased(character.TemporaryStatTypeRepeatEffect)
	newAndIncNonDiseased(character.TemporaryStatTypeExpBuffRate)
	newAndIncNonDiseased(character.TemporaryStatTypeStopPortion)
	newAndIncNonDiseased(character.TemporaryStatTypeStopMotion)
	newAndIncNonDiseased(character.TemporaryStatTypeFear)
	newAndIncNonDiseased(character.TemporaryStatTypeEvanSlow)
	newAndIncNonDiseased(character.TemporaryStatTypeMagicShield)
	newAndIncNonDiseased(character.TemporaryStatTypeMagicResist)
	newAndIncNonDiseased(character.TemporaryStatTypeSoulStone)
	if (tenant.Region() == "GMS" && tenant.MajorVersion() > 83) || tenant.Region() == "JMS" {
		newAndIncNonDiseased(character.TemporaryStatTypeFlying)
		newAndIncNonDiseased(character.TemporaryStatTypeFrozen)
		newAndIncNonDiseased(character.TemporaryStatTypeAssistCharge)
		newAndIncNonDiseased(character.TemporaryStatTypeMirrorImage)
		newAndIncNonDiseased(character.TemporaryStatTypeSuddenDeath)
		newAndIncNonDiseased(character.TemporaryStatTypeNotDamaged)
		newAndIncNonDiseased(character.TemporaryStatTypeFinalCut)
		newAndIncNonDiseased(character.TemporaryStatTypeThornsEffect)
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowAttackDamage)
		newAndIncNonDiseased(character.TemporaryStatTypeWildDamageUp)
		newAndIncNonDiseased(character.TemporaryStatTypeMine)
		newAndIncNonDiseased(character.TemporaryStatTypeEMHP)
		newAndIncNonDiseased(character.TemporaryStatTypeEMMP)
		newAndIncNonDiseased(character.TemporaryStatTypeEPAD)
		newAndIncNonDiseased(character.TemporaryStatTypeEPPD)
		newAndIncNonDiseased(character.TemporaryStatTypeEMDD)
		newAndIncNonDiseased(character.TemporaryStatTypeGuard)
		newAndIncNonDiseased(character.TemporaryStatTypeSafetyDamage)
		newAndIncNonDiseased(character.TemporaryStatTypeSafetyAbsorb)
		newAndIncNonDiseased(character.TemporaryStatTypeCyclone)
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowCritical)
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowMaxMP)
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowDefense)
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowEvasion)
		newAndIncNonDiseased(character.TemporaryStatTypeConversion)
		newAndIncNonDiseased(character.TemporaryStatTypeRevive)
		newAndIncNonDiseased(character.TemporaryStatTypeSneak)

		newAndIncNonDiseased(character.TemporaryStatTypeUnknown)
	}
	newAndIncNonDiseased(character.TemporaryStatTypeEnergyCharge)
	newAndIncNonDiseased(character.TemporaryStatTypeDashSpeed)
	newAndIncNonDiseased(character.TemporaryStatTypeDashJump)
	newAndIncNonDiseased(character.TemporaryStatTypeMonsterRiding)
	newAndIncNonDiseased(character.TemporaryStatTypeSpeedInfusion)
	newAndIncNonDiseased(character.TemporaryStatTypeHomingBeacon)
	newAndIncDiseased(character.TemporaryStatTypeUndead)

	return func(name character.TemporaryStatType) (CharacterTemporaryStatType, error) {
		if val, ok := set[name]; ok {
			return val, nil
		}
		return CharacterTemporaryStatType{}, errors.New("character temporary stat type not found")
	}
}

type CharacterTemporaryStatValue struct {
	sourceId  uint32
	value     int32
	expiresAt time.Time
}

func (v CharacterTemporaryStatValue) Value() int32 {
	return v.value
}

func (v CharacterTemporaryStatValue) SourceId() uint32 {
	return v.sourceId
}

func (v CharacterTemporaryStatValue) ExpiresAt() time.Time {
	return v.expiresAt
}

type CharacterTemporaryStatBase struct {
	bDynamicTermSet bool
	nOption         int32
	rOption         int32
	tLastUpdated    int64
	usExpireItem    int16
}

func NewCharacterTemporaryStatBase(bDynamicTermSet bool) CharacterTemporaryStatBase {
	return CharacterTemporaryStatBase{
		tLastUpdated:    time.Now().Unix(),
		bDynamicTermSet: bDynamicTermSet,
	}
}

func writeTime(t int64) func(w *response.Writer) {
	return func(w *response.Writer) {
		cur := time.Now().Unix()
		interval := false
		if t >= cur {
			t -= cur
		} else {
			interval = true
			t = cur - t
		}
		t /= 1000
		w.WriteBool(interval)
		w.WriteInt32(int32(t))
	}
}

func (m CharacterTemporaryStatBase) Encode(_ logrus.FieldLogger, t tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt32(m.nOption)
		w.WriteInt32(m.rOption)
		writeTime(m.tLastUpdated)(w)
		if m.bDynamicTermSet {
			w.WriteInt16(m.usExpireItem)
		}
	}
}

type SpeedInfusionTemporaryStat struct {
	CharacterTemporaryStatBase
	tCurrentTime int32
}

func (m SpeedInfusionTemporaryStat) Encode(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		m.CharacterTemporaryStatBase.Encode(l, t, options)(w)
		writeTime(int64(m.tCurrentTime))(w)
		w.WriteInt16(m.usExpireItem)
	}
}

func NewSpeedInfusionTemporaryStat() SpeedInfusionTemporaryStat {
	return SpeedInfusionTemporaryStat{
		CharacterTemporaryStatBase: CharacterTemporaryStatBase{
			bDynamicTermSet: false,
			nOption:         0,
			rOption:         0,
			tLastUpdated:    time.Now().Unix(),
			usExpireItem:    0,
		},
		tCurrentTime: 0,
	}
}

type GuidedBulletTemporaryStat struct {
	CharacterTemporaryStatBase
	dwMobId uint32
}

func (m GuidedBulletTemporaryStat) Encode(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		m.CharacterTemporaryStatBase.Encode(l, t, options)(w)
		w.WriteInt(m.dwMobId)
	}
}

func NewGuidedBulletTemporaryStat() GuidedBulletTemporaryStat {
	return GuidedBulletTemporaryStat{
		CharacterTemporaryStatBase: CharacterTemporaryStatBase{
			bDynamicTermSet: false,
			nOption:         0,
			rOption:         0,
			tLastUpdated:    time.Now().Unix(),
			usExpireItem:    0,
		},
		dwMobId: 0,
	}
}

type CharacterTemporaryStat struct {
	stats map[CharacterTemporaryStatType]CharacterTemporaryStatValue
}

func NewCharacterTemporaryStat() *CharacterTemporaryStat {
	return &CharacterTemporaryStat{
		stats: make(map[CharacterTemporaryStatType]CharacterTemporaryStatValue),
	}
}

func (m *CharacterTemporaryStat) AddStat(l logrus.FieldLogger) func(t tenant.Model) func(name string, sourceId uint32, amount int32, expiresAt time.Time) {
	return func(t tenant.Model) func(name string, sourceId uint32, amount int32, expiresAt time.Time) {
		return func(name string, sourceId uint32, amount int32, expiresAt time.Time) {
			st, err := CharacterTemporaryStatTypeByName(t)(character.TemporaryStatType(name))
			if err != nil {
				l.WithError(err).Errorf("Attempting to add buff [%s], but cannot find it.", name)
				return
			}
			v := CharacterTemporaryStatValue{
				sourceId:  sourceId,
				value:     amount,
				expiresAt: expiresAt,
			}
			if e, ok := m.stats[st]; ok {
				if v.Value() > e.Value() {
					m.stats[st] = v
				}
			} else {
				m.stats[st] = v
			}
		}
	}
}

func (m *CharacterTemporaryStat) EncodeMask(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		mask := tool.Uint128{}
		applyMask := func(name character.TemporaryStatType) {
			if val, err := CharacterTemporaryStatTypeByName(t)(name); err == nil {
				mask = mask.Or(val.mask)
			}
		}
		applyMask(character.TemporaryStatTypeEnergyCharge)
		applyMask(character.TemporaryStatTypeDashSpeed)
		applyMask(character.TemporaryStatTypeDashJump)
		applyMask(character.TemporaryStatTypeMonsterRiding)
		applyMask(character.TemporaryStatTypeSpeedInfusion)
		applyMask(character.TemporaryStatTypeHomingBeacon)
		applyMask(character.TemporaryStatTypeUndead)

		for s := range m.stats {
			mask = mask.Or(s.mask)
		}

		w.WriteInt(uint32(mask.H >> 32))
		w.WriteInt(uint32(mask.H & 0xFFFFFFFF))
		w.WriteInt(uint32(mask.L >> 32))
		w.WriteInt(uint32(mask.L & 0xFFFFFFFF))
	}
}

func (m *CharacterTemporaryStat) Encode(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		m.EncodeMask(l, t, options)(w)

		keys := make([]CharacterTemporaryStatType, 0)
		for k := range m.stats {
			keys = append(keys, k)
		}

		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Shift() < keys[j].Shift()
		})

		// Create a slice of values sorted by the keys' index
		sortedValues := make([]CharacterTemporaryStatValue, 0)
		for _, k := range keys {
			sortedValues = append(sortedValues, m.stats[k])
		}

		for _, v := range sortedValues {
			w.WriteInt16(int16(v.Value()))
			w.WriteInt(v.SourceId())
			et := int32(v.ExpiresAt().Sub(time.Now()).Milliseconds())
			w.WriteInt32(et)
		}

		w.WriteByte(0) // nDefenseAtt
		w.WriteByte(0) // nDefenseState

		var baseTemporaryStats = m.getBaseTemporaryStats()
		for _, bts := range baseTemporaryStats {
			bts.Encode(l, t, options)(w)
		}
	}
}

func (m *CharacterTemporaryStat) getBaseTemporaryStats() []Encoder {
	var list = make([]Encoder, 0)
	list = append(list, NewCharacterTemporaryStatBase(true)) // Energy Charge 15
	list = append(list, NewCharacterTemporaryStatBase(true)) // Dash Speed 15
	list = append(list, NewCharacterTemporaryStatBase(true)) // Dash Jump 15
	// TODO look up actual buff values if riding mount.
	list = append(list, NewCharacterTemporaryStatBase(false)) // Monster Riding 13
	list = append(list, NewSpeedInfusionTemporaryStat())      // 17
	list = append(list, NewGuidedBulletTemporaryStat())       // 17
	list = append(list, NewCharacterTemporaryStatBase(true))  // Undead 15
	return list
}
