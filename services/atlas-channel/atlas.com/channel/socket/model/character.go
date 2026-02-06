package model

import (
	"atlas-channel/tool"
	"errors"
	"sort"
	"time"

	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type CharacterTemporaryStatType struct {
	name               character.TemporaryStatType
	shift              uint
	mask               tool.Uint128
	disease            bool
	foreignValueWriter ForeignValueWriter
}

func (t CharacterTemporaryStatType) Shift() uint {
	return t.shift
}

func NewCharacterTemporaryStatType(name character.TemporaryStatType, shift uint, disease bool, foreignValueWriter ForeignValueWriter) CharacterTemporaryStatType {
	mask := tool.Uint128{L: 1}.ShiftLeft(shift)
	return CharacterTemporaryStatType{
		name:               name,
		shift:              shift,
		mask:               mask,
		disease:            disease,
		foreignValueWriter: foreignValueWriter,
	}
}

func CharacterTemporaryStatTypeByName(t tenant.Model) func(name character.TemporaryStatType) (CharacterTemporaryStatType, error) {
	var shift uint = 0
	set := make(map[character.TemporaryStatType]CharacterTemporaryStatType)

	funcCallNewAndInc := func(disease bool) func(name character.TemporaryStatType) func(foreignValueWriter ForeignValueWriter) {
		return func(name character.TemporaryStatType) func(foreignValueWriter ForeignValueWriter) {
			return func(foreignValueWriter ForeignValueWriter) {
				set[name] = NewCharacterTemporaryStatType(name, shift, disease, foreignValueWriter)
				shift += 1
			}
		}
	}
	newAndIncDiseased := funcCallNewAndInc(true)
	newAndIncNonDiseased := funcCallNewAndInc(false)

	newAndIncNonDiseased(character.TemporaryStatTypeWeaponAttack)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeWeaponDefense)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeMagicAttack)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeMagicDefense)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeAccuracy)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeAvoidability)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeHands)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeSpeed)(ValueAsByteForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeJump)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeMagicGuard)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeDarkSight)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeBooster)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypePowerGuard)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeHyperBodyHP)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeHyperBodyMP)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeInvincible)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeSoulArrow)(NoOpForeignValueWriter)
	newAndIncDiseased(character.TemporaryStatTypeStun)(ValueAsIntForeignValueWriter)
	newAndIncDiseased(character.TemporaryStatTypePoison)(ValueSourceLevelForeignValueWriter)
	newAndIncDiseased(character.TemporaryStatTypeSeal)(ValueAsIntForeignValueWriter)
	newAndIncDiseased(character.TemporaryStatTypeDarkness)(ValueAsIntForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeCombo)(ValueAsByteForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeWhiteKnightCharge)(ValueAsIntForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeDragonBlood)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeHolySymbol)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeMesoUp)(NoOpForeignValueWriter)
	if t.Region() == "GMS" && t.MajorVersion() > 83 {
		newAndIncNonDiseased(character.TemporaryStatTypeShadowPartner)(LevelSourceForeignValueWriter)
	} else {
		newAndIncNonDiseased(character.TemporaryStatTypeShadowPartner)(NoOpForeignValueWriter)
	}
	newAndIncNonDiseased(character.TemporaryStatTypePickPocket)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeMesoGuard)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeThaw)(NoOpForeignValueWriter)
	newAndIncDiseased(character.TemporaryStatTypeWeaken)(ValueAsIntForeignValueWriter)
	newAndIncDiseased(character.TemporaryStatTypeCurse)(ValueAsIntForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeSlow)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeMorph)(ValueAsShortForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeRecovery)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeMapleWarrior)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeStance)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeSharpEyes)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeManaReflection)(NoOpForeignValueWriter)
	newAndIncDiseased(character.TemporaryStatTypeSeduce)(LevelSourceForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeShadowClaw)(ValueAsIntForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeInfinity)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeHolyShield)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeHamstring)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeBlind)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeConcentrate)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeBanMap)(ValueAsIntForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeEchoOfHero)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeMesoUpByItem)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeGhostMorph)(ValueAsShortForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeBarrier)(ValueAsIntForeignValueWriter)
	newAndIncDiseased(character.TemporaryStatTypeConfuse)(LevelSourceForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeItemUpByItem)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeRespectPImmune)(ValueAsIntForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeRespectMImmune)(ValueAsIntForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeDefenseAttack)(ValueAsIntForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeDefenseState)(ValueAsIntForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeIncreaseEffectHpPotion)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeIncreaseEffectMpPotion)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeBerserkFury)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeDivineBody)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeSpark)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeDojangShield)(ValueAsIntForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeSoulMasterFinal)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeWindBreakerFinal)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeElementalReset)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeWindWalk)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeEventRate)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeAranCombo)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeComboDrain)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeComboBarrier)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeBodyPressure)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeSmartKnockBack)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeRepeatEffect)(ValueAsIntForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeExpBuffRate)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeStopPortion)(ValueAsIntForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeStopMotion)(ValueAsIntForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeFear)(ValueAsIntForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeEvanSlow)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeMagicShield)(ValueAsIntForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeMagicResist)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeSoulStone)(NoOpForeignValueWriter)
	if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
		newAndIncNonDiseased(character.TemporaryStatTypeFlying)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeFrozen)(ValueAsIntForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeAssistCharge)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeMirrorImage)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeSuddenDeath)(ValueAsIntForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeNotDamaged)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeFinalCut)(ValueAsIntForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeThornsEffect)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowAttackDamage)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeWildDamageUp)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeMine)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeEMHP)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeEMMP)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeEPAD)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeEPPD)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeEMDD)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeGuard)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeSafetyDamage)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeSafetyAbsorb)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeCyclone)(ValueAsByteForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowCritical)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowMaxMP)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowDefense)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeSwallowEvasion)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeConversion)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeRevive)(NoOpForeignValueWriter)
		newAndIncNonDiseased(character.TemporaryStatTypeSneak)(NoOpForeignValueWriter)

		newAndIncNonDiseased(character.TemporaryStatTypeUnknown)(NoOpForeignValueWriter)
	}
	newAndIncNonDiseased(character.TemporaryStatTypeEnergyCharge)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeDashSpeed)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeDashJump)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeMonsterRiding)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeSpeedInfusion)(NoOpForeignValueWriter)
	newAndIncNonDiseased(character.TemporaryStatTypeHomingBeacon)(NoOpForeignValueWriter)
	newAndIncDiseased(character.TemporaryStatTypeUndead)(NoOpForeignValueWriter)

	return func(name character.TemporaryStatType) (CharacterTemporaryStatType, error) {
		if val, ok := set[name]; ok {
			return val, nil
		}
		return CharacterTemporaryStatType{}, errors.New("character temporary stat type not found")
	}
}

type ForeignValueWriter func(v CharacterTemporaryStatValue) func(w *response.Writer)

func NoOpForeignValueWriter(_ CharacterTemporaryStatValue) func(w *response.Writer) {
	return func(_ *response.Writer) {
	}
}

func ValueAsByteForeignValueWriter(v CharacterTemporaryStatValue) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt8(int8(v.Value()))
	}
}

func ValueAsShortForeignValueWriter(v CharacterTemporaryStatValue) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt16(int16(v.Value()))
	}
}

func ValueAsIntForeignValueWriter(v CharacterTemporaryStatValue) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt32(v.Value())
	}
}

func LevelSourceForeignValueWriter(v CharacterTemporaryStatValue) func(w *response.Writer) {
	return func(w *response.Writer) {
		// TODO
		w.WriteInt16(int16(1))
		w.WriteInt16(int16(v.SourceId()))
	}
}

func ValueSourceLevelForeignValueWriter(v CharacterTemporaryStatValue) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt16(int16(v.Value()))
		// TODO
		w.WriteInt16(int16(1))
		w.WriteInt16(int16(v.SourceId()))
	}
}

type CharacterTemporaryStatValue struct {
	statType  CharacterTemporaryStatType
	sourceId  int32
	value     int32
	expiresAt time.Time
}

func (v CharacterTemporaryStatValue) Value() int32 {
	return v.value
}

func (v CharacterTemporaryStatValue) SourceId() int32 {
	return v.sourceId
}

func (v CharacterTemporaryStatValue) ExpiresAt() time.Time {
	return v.expiresAt
}

func (v CharacterTemporaryStatValue) Write(w *response.Writer) {
	v.statType.foreignValueWriter(v)(w)
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
	stats map[character.TemporaryStatType]CharacterTemporaryStatValue
}

func NewCharacterTemporaryStat() *CharacterTemporaryStat {
	return &CharacterTemporaryStat{
		stats: make(map[character.TemporaryStatType]CharacterTemporaryStatValue),
	}
}

func (m *CharacterTemporaryStat) AddStat(l logrus.FieldLogger) func(t tenant.Model) func(n string, sourceId int32, amount int32, expiresAt time.Time) {
	return func(t tenant.Model) func(n string, sourceId int32, amount int32, expiresAt time.Time) {
		return func(n string, sourceId int32, amount int32, expiresAt time.Time) {
			name := character.TemporaryStatType(n)
			st, err := CharacterTemporaryStatTypeByName(t)(name)
			if err != nil {
				l.WithError(err).Errorf("Attempting to add buff [%s], but cannot find it.", name)
				return
			}
			v := CharacterTemporaryStatValue{
				statType:  st,
				sourceId:  sourceId,
				value:     amount,
				expiresAt: expiresAt,
			}
			if e, ok := m.stats[name]; ok {
				if v.Value() > e.Value() {
					m.stats[name] = v
				}
			} else {
				m.stats[name] = v
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

		for _, v := range m.stats {
			mask = mask.Or(v.statType.mask)
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
		for _, v := range m.stats {
			keys = append(keys, v.statType)
		}

		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Shift() < keys[j].Shift()
		})

		// Create a slice of values sorted by the keys' index
		sortedValues := make([]CharacterTemporaryStatValue, 0)
		for _, k := range keys {
			sortedValues = append(sortedValues, m.stats[k.name])
		}

		for _, v := range sortedValues {
			w.WriteInt16(int16(v.Value()))
			w.WriteInt32(v.SourceId())
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

func (m *CharacterTemporaryStat) EncodeForeign(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		m.EncodeMask(l, t, options)(w)

		keys := make([]CharacterTemporaryStatType, 0)
		for _, v := range m.stats {
			keys = append(keys, v.statType)
		}

		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Shift() < keys[j].Shift()
		})

		// Create a slice of values sorted by the keys' index
		sortedValues := make([]CharacterTemporaryStatValue, 0)
		for _, v := range keys {
			sortedValues = append(sortedValues, m.stats[v.name])
		}

		for _, v := range sortedValues {
			v.Write(w)
			//w.WriteInt32(v.Value())
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
