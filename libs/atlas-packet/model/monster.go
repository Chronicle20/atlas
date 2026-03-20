package model

import (
	"context"
	"errors"
	"math"
	"sort"
	"time"

	"github.com/Chronicle20/atlas-constants/monster"
	"github.com/Chronicle20/atlas-packet/tool"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type MonsterAppearType int8

const (
	MonsterAppearTypeNormal    MonsterAppearType = -1
	MonsterAppearTypeRegen     MonsterAppearType = -2
	MonsterAppearTypeRevived   MonsterAppearType = -3
	MonsterAppearTypeSuspended MonsterAppearType = -4
	MonsterAppearTypeDelay     MonsterAppearType = -5
	// Effect.wz/Summon.img
	MonsterAppearTypeBalrog              MonsterAppearType = 0
	MonsterAppearTypeSmoke               MonsterAppearType = 1
	MonsterAppearTypeWerewolf            MonsterAppearType = 2
	MonsterAppearTypeKingSlimeMinion     MonsterAppearType = 3
	MonsterAppearTypeSummoningRock       MonsterAppearType = 4
	MonsterAppearTypeEyeOfHorus          MonsterAppearType = 5
	MonsterAppearTypeBlueStars           MonsterAppearType = 6
	MonsterAppearTypeSmoke2              MonsterAppearType = 7
	MonsterAppearTypeTheBoss             MonsterAppearType = 8
	MonsterAppearTypeGrimPhantomBlack    MonsterAppearType = 9
	MonsterAppearTypeGrimPhantomBlue     MonsterAppearType = 10
	MonsterAppearTypeThorn               MonsterAppearType = 11
	MonsterAppearTypeUnknown             MonsterAppearType = 12
	MonsterAppearTypeFrankenstein        MonsterAppearType = 13
	MonsterAppearTypeFrankensteinEnraged MonsterAppearType = 14
	MonsterAppearTypeOrbit               MonsterAppearType = 15
	MonsterAppearTypeHiver               MonsterAppearType = 16
	MonsterAppearTypeSmoke3              MonsterAppearType = 17
	MonsterAppearTypeSmoke4              MonsterAppearType = 18
	MonsterAppearTypePrimeMinister       MonsterAppearType = 19
	MonsterAppearTypePrimeMinister2      MonsterAppearType = 23
	MonsterAppearTypeOlivia              MonsterAppearType = 24
	MonsterAppearTypeWingedEvilStump     MonsterAppearType = 25
	MonsterAppearTypeWingedEvilStump2    MonsterAppearType = 26
	MonsterAppearTypeApsu                MonsterAppearType = 27
	MonsterAppearTypeBlackFluid          MonsterAppearType = 28
	MonsterAppearTypeHiver2              MonsterAppearType = 29
	MonsterAppearTypeDragonRider         MonsterAppearType = 30
)

type MonsterTemporaryStatType struct {
	name  monster.TemporaryStatType
	shift uint
	mask  tool.Uint128
}

func (t MonsterTemporaryStatType) Name() monster.TemporaryStatType {
	return t.name
}

func (t MonsterTemporaryStatType) Shift() uint {
	return t.shift
}

func (t MonsterTemporaryStatType) Mask() tool.Uint128 {
	return t.mask
}

func NewMonsterTemporaryStatType(name monster.TemporaryStatType, shift uint) MonsterTemporaryStatType {
	mask := tool.Uint128{L: 1}.ShiftLeft(shift)
	return MonsterTemporaryStatType{
		name:  name,
		shift: shift,
		mask:  mask,
	}
}

func MonsterTemporaryStatTypeByName(t tenant.Model) func(name monster.TemporaryStatType) (MonsterTemporaryStatType, error) {
	var shift uint = 0
	set := make(map[monster.TemporaryStatType]MonsterTemporaryStatType)

	funcCallNewAndInc := func(name monster.TemporaryStatType) {
		set[name] = NewMonsterTemporaryStatType(name, shift)
		shift += 1
	}
	funcCallNewAndInc(monster.TemporaryStatTypeWeaponAttack)
	funcCallNewAndInc(monster.TemporaryStatTypeWeaponDefense)
	funcCallNewAndInc(monster.TemporaryStatTypeMagicAttack)
	funcCallNewAndInc(monster.TemporaryStatTypeMagicDefense)
	funcCallNewAndInc(monster.TemporaryStatTypeAccuracy)
	funcCallNewAndInc(monster.TemporaryStatTypeAvoidability)
	funcCallNewAndInc(monster.TemporaryStatTypeSpeed)
	funcCallNewAndInc(monster.TemporaryStatTypeStun)
	funcCallNewAndInc(monster.TemporaryStatTypeFrozen)
	funcCallNewAndInc(monster.TemporaryStatTypePoison)
	funcCallNewAndInc(monster.TemporaryStatTypeSeal)
	funcCallNewAndInc(monster.TemporaryStatTypeDarkness)
	funcCallNewAndInc(monster.TemporaryStatTypePowerUp)
	funcCallNewAndInc(monster.TemporaryStatTypeMagicUp)
	funcCallNewAndInc(monster.TemporaryStatTypePowerGuardUp)
	funcCallNewAndInc(monster.TemporaryStatTypeMagicGuardUp)
	funcCallNewAndInc(monster.TemporaryStatTypeDoom)
	funcCallNewAndInc(monster.TemporaryStatTypeWeb)
	funcCallNewAndInc(monster.TemporaryStatTypeWeaponAttackImmune)
	funcCallNewAndInc(monster.TemporaryStatTypeMagicAttackImmune)
	funcCallNewAndInc(monster.TemporaryStatTypeShowdown)
	funcCallNewAndInc(monster.TemporaryStatTypeHardSkin)
	funcCallNewAndInc(monster.TemporaryStatTypeAmbush)
	funcCallNewAndInc(monster.TemporaryStatTypeDamagedElemAttr)
	funcCallNewAndInc(monster.TemporaryStatTypeVenom)
	funcCallNewAndInc(monster.TemporaryStatTypeBlind)
	funcCallNewAndInc(monster.TemporaryStatTypeSealSkill)
	funcCallNewAndInc(monster.TemporaryStatTypeBurned)
	funcCallNewAndInc(monster.TemporaryStatTypeDazzle)
	funcCallNewAndInc(monster.TemporaryStatTypeWeaponCounter)
	funcCallNewAndInc(monster.TemporaryStatTypeMagicCounter)
	funcCallNewAndInc(monster.TemporaryStatTypeDisable)
	funcCallNewAndInc(monster.TemporaryStatTypeRiseByToss)
	funcCallNewAndInc(monster.TemporaryStatTypeBodyPressure)
	funcCallNewAndInc(monster.TemporaryStatTypeWeakness)
	funcCallNewAndInc(monster.TemporaryStatTypeTimeBomb)
	funcCallNewAndInc(monster.TemporaryStatTypeMagicCrash)
	funcCallNewAndInc(monster.TemporaryStatTypeHealByDamage)

	return func(name monster.TemporaryStatType) (MonsterTemporaryStatType, error) {
		if val, ok := set[name]; ok {
			return val, nil
		}
		return MonsterTemporaryStatType{}, errors.New("monster temporary stat type not found")
	}
}

type MonsterTemporaryStatValue struct {
	statType    MonsterTemporaryStatType
	sourceId    int32
	sourceLevel int32
	value       int32
	expiresAt   time.Time
}

func (m MonsterTemporaryStatValue) StatType() MonsterTemporaryStatType {
	return m.statType
}

func (m MonsterTemporaryStatValue) SourceId() int32 {
	return m.sourceId
}

func (m MonsterTemporaryStatValue) SourceLevel() int32 {
	return m.sourceLevel
}

func (m MonsterTemporaryStatValue) Value() int32 {
	return m.value
}

func (m MonsterTemporaryStatValue) ExpiresAt() time.Time {
	return m.expiresAt
}

type MonsterBurnedInfo struct {
	characterId uint32
	skillId     uint32
	damage      uint32
	interval    uint32
	end         uint32
	dotCount    uint32
}

func (m MonsterBurnedInfo) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteInt(m.skillId)
		w.WriteInt(m.damage)
		w.WriteInt(m.interval)
		w.WriteInt(m.end)
		w.WriteInt(m.dotCount)
		return w.Bytes()
	}
}

func (m *MonsterBurnedInfo) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.skillId = r.ReadUint32()
		m.damage = r.ReadUint32()
		m.interval = r.ReadUint32()
		m.end = r.ReadUint32()
		m.dotCount = r.ReadUint32()
	}
}

type MonsterTemporaryStat struct {
	burnedInfo []MonsterBurnedInfo
	stats      map[monster.TemporaryStatType]MonsterTemporaryStatValue
}

func (m *MonsterTemporaryStat) Mask() tool.Uint128 {
	mask := tool.Uint128{}
	for _, v := range m.stats {
		mask = mask.Or(v.StatType().Mask())
	}
	return mask
}

func (m *MonsterTemporaryStat) IsMovementAffectingStat(t tenant.Model) bool {
	lookup := MonsterTemporaryStatTypeByName(t)
	filter := tool.Uint128{}
	for _, name := range []monster.TemporaryStatType{
		monster.TemporaryStatTypeSpeed,
		monster.TemporaryStatTypeStun,
		monster.TemporaryStatTypeFrozen,
		monster.TemporaryStatTypeDoom,
		monster.TemporaryStatTypeRiseByToss,
	} {
		if st, err := lookup(name); err == nil {
			filter = filter.Or(st.Mask())
		}
	}
	result := m.Mask().And(filter)
	return result.H != 0 || result.L != 0
}

func (m *MonsterTemporaryStat) EncodeMask(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		mask := m.Mask()
		w.WriteInt(uint32(mask.H >> 32))
		w.WriteInt(uint32(mask.H & 0xFFFFFFFF))
		w.WriteInt(uint32(mask.L >> 32))
		w.WriteInt(uint32(mask.L & 0xFFFFFFFF))
	}
}

func (m *MonsterTemporaryStat) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		m.EncodeMask(l, t, options)(w)

		keys := make([]MonsterTemporaryStatType, 0)
		for _, v := range m.stats {
			keys = append(keys, v.statType)
		}

		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Shift() < keys[j].Shift()
		})

		sortedValues := make([]MonsterTemporaryStatValue, 0)
		for _, k := range keys {
			sortedValues = append(sortedValues, m.stats[k.name])
		}

		weaponCounter := int32(-1)
		magicCounter := int32(-1)
		for k, v := range sortedValues {
			tst := sortedValues[k].StatType().Name()
			if tst == monster.TemporaryStatTypeBurned {
				w.WriteInt(uint32(len(m.burnedInfo)))
				for _, b := range m.burnedInfo {
					w.WriteByteArray(b.Encode(l, ctx)(options))
				}
				continue
			}
			if tst == monster.TemporaryStatTypeDisable {
				w.WriteBool(false) // bInvincible
				w.WriteBool(false) // bDisable
				continue
			}

			if tst == monster.TemporaryStatTypeWeaponCounter {
				weaponCounter = v.Value()
			}
			if tst == monster.TemporaryStatTypeMagicCounter {
				magicCounter = v.Value()
			}
			w.WriteInt16(int16(v.Value()))
			w.WriteInt16(int16(v.SourceId()))
			w.WriteInt16(int16(v.SourceLevel()))
			w.WriteInt16(monsterStatExpiry(v.ExpiresAt()))
		}

		if weaponCounter != -1 {
			w.WriteInt32(weaponCounter)
		}
		if magicCounter != -1 {
			w.WriteInt32(magicCounter)
		}
		if weaponCounter != -1 || magicCounter != -1 {
			w.WriteInt32(int32(math.Max(float64(weaponCounter), float64(magicCounter))))
		}
		return w.Bytes()
	}
}

func (m *MonsterTemporaryStat) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		var mask tool.Uint128
		h1 := uint64(r.ReadUint32())
		h2 := uint64(r.ReadUint32())
		l1 := uint64(r.ReadUint32())
		l2 := uint64(r.ReadUint32())
		mask.H = (h1 << 32) | h2
		mask.L = (l1 << 32) | l2

		lookup := MonsterTemporaryStatTypeByName(t)

		allStats := []monster.TemporaryStatType{
			monster.TemporaryStatTypeWeaponAttack,
			monster.TemporaryStatTypeWeaponDefense,
			monster.TemporaryStatTypeMagicAttack,
			monster.TemporaryStatTypeMagicDefense,
			monster.TemporaryStatTypeAccuracy,
			monster.TemporaryStatTypeAvoidability,
			monster.TemporaryStatTypeSpeed,
			monster.TemporaryStatTypeStun,
			monster.TemporaryStatTypeFrozen,
			monster.TemporaryStatTypePoison,
			monster.TemporaryStatTypeSeal,
			monster.TemporaryStatTypeDarkness,
			monster.TemporaryStatTypePowerUp,
			monster.TemporaryStatTypeMagicUp,
			monster.TemporaryStatTypePowerGuardUp,
			monster.TemporaryStatTypeMagicGuardUp,
			monster.TemporaryStatTypeDoom,
			monster.TemporaryStatTypeWeb,
			monster.TemporaryStatTypeWeaponAttackImmune,
			monster.TemporaryStatTypeMagicAttackImmune,
			monster.TemporaryStatTypeShowdown,
			monster.TemporaryStatTypeHardSkin,
			monster.TemporaryStatTypeAmbush,
			monster.TemporaryStatTypeDamagedElemAttr,
			monster.TemporaryStatTypeVenom,
			monster.TemporaryStatTypeBlind,
			monster.TemporaryStatTypeSealSkill,
			monster.TemporaryStatTypeBurned,
			monster.TemporaryStatTypeDazzle,
			monster.TemporaryStatTypeWeaponCounter,
			monster.TemporaryStatTypeMagicCounter,
			monster.TemporaryStatTypeDisable,
			monster.TemporaryStatTypeRiseByToss,
			monster.TemporaryStatTypeBodyPressure,
			monster.TemporaryStatTypeWeakness,
			monster.TemporaryStatTypeTimeBomb,
			monster.TemporaryStatTypeMagicCrash,
			monster.TemporaryStatTypeHealByDamage,
		}

		m.stats = make(map[monster.TemporaryStatType]MonsterTemporaryStatValue)
		weaponCounter := int32(-1)
		magicCounter := int32(-1)
		for _, name := range allStats {
			st, err := lookup(name)
			if err != nil {
				continue
			}
			check := mask.And(st.Mask())
			if check.H == 0 && check.L == 0 {
				continue
			}
			if name == monster.TemporaryStatTypeBurned {
				count := r.ReadUint32()
				m.burnedInfo = make([]MonsterBurnedInfo, count)
				for i := uint32(0); i < count; i++ {
					m.burnedInfo[i].Decode(l, ctx)(r, options)
				}
				continue
			}
			if name == monster.TemporaryStatTypeDisable {
				_ = r.ReadBool() // bInvincible
				_ = r.ReadBool() // bDisable
				m.stats[name] = MonsterTemporaryStatValue{statType: st}
				continue
			}
			value := int32(r.ReadInt16())
			sourceId := int32(r.ReadInt16())
			sourceLevel := int32(r.ReadInt16())
			_ = r.ReadInt16() // expiry
			m.stats[name] = MonsterTemporaryStatValue{
				statType:    st,
				sourceId:    sourceId,
				sourceLevel: sourceLevel,
				value:       value,
			}
			if name == monster.TemporaryStatTypeWeaponCounter {
				weaponCounter = value
			}
			if name == monster.TemporaryStatTypeMagicCounter {
				magicCounter = value
			}
		}

		if weaponCounter != -1 {
			_ = r.ReadInt32()
		}
		if magicCounter != -1 {
			_ = r.ReadInt32()
		}
		if weaponCounter != -1 || magicCounter != -1 {
			_ = r.ReadInt32()
		}
	}
}

func monsterStatExpiry(_ time.Time) int16 {
	return -1
}

func NewMonsterTemporaryStat() *MonsterTemporaryStat {
	return &MonsterTemporaryStat{
		stats: make(map[monster.TemporaryStatType]MonsterTemporaryStatValue),
	}
}

func (m *MonsterTemporaryStat) AddStat(l logrus.FieldLogger) func(t tenant.Model) func(n string, sourceId uint32, sourceLevel uint32, amount int32, expiresAt time.Time) {
	return func(t tenant.Model) func(n string, sourceId uint32, sourceLevel uint32, amount int32, expiresAt time.Time) {
		return func(n string, sourceId uint32, sourceLevel uint32, amount int32, expiresAt time.Time) {
			name := monster.TemporaryStatType(n)
			st, err := MonsterTemporaryStatTypeByName(t)(name)
			if err != nil {
				l.WithError(err).Errorf("Attempting to add buff [%s], but cannot find it.", name)
				return
			}
			v := MonsterTemporaryStatValue{
				statType:    st,
				sourceId:    int32(sourceId),
				sourceLevel: int32(sourceLevel),
				value:       amount,
				expiresAt:   expiresAt,
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

func (m *MonsterTemporaryStat) SetTemporaryStat(stat *MonsterTemporaryStat) {
	*m = *stat
}

type MonsterModel struct {
	monsterTemporaryStat MonsterTemporaryStat
	x                    int16
	y                    int16
	moveAction           byte
	foothold             int16
	homeFoothold         int16
	appearType           MonsterAppearType
	appearTypeOption     uint32
	team                 int8
	effectItemId         uint32
	phase                uint32
}

func NewMonster(x int16, y int16, stance byte, fh int16, appearType MonsterAppearType, team int8) MonsterModel {
	return MonsterModel{
		monsterTemporaryStat: MonsterTemporaryStat{},
		x:                    x,
		y:                    y,
		moveAction:           stance,
		foothold:             0,
		homeFoothold:         fh,
		appearType:           appearType,
		appearTypeOption:     0,
		team:                 team,
		effectItemId:         0,
		phase:                0,
	}
}

func (m *MonsterModel) SetTemporaryStat(stat *MonsterTemporaryStat) {
	m.monsterTemporaryStat = *stat
}

func (m *MonsterModel) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			w.WriteByteArray(m.monsterTemporaryStat.Encode(l, ctx)(options))
		}
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		w.WriteByte(m.moveAction)
		w.WriteInt16(m.foothold)
		w.WriteInt16(m.homeFoothold)
		w.WriteInt8(int8(m.appearType))
		if m.appearType == MonsterAppearTypeRevived || m.appearType >= 0 {
			w.WriteInt(m.appearTypeOption)
		}
		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			w.WriteInt8(m.team)
			w.WriteInt(m.effectItemId)
			if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
				w.WriteInt(m.phase)
			}
		} else {
			// TODO proper temp stat encoding for GMS v12
			w.WriteInt(0)
		}
		return w.Bytes()
	}
}

func (m *MonsterModel) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			m.monsterTemporaryStat.Decode(l, ctx)(r, options)
		}
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
		m.moveAction = r.ReadByte()
		m.foothold = r.ReadInt16()
		m.homeFoothold = r.ReadInt16()
		m.appearType = MonsterAppearType(r.ReadInt8())
		if m.appearType == MonsterAppearTypeRevived || m.appearType >= 0 {
			m.appearTypeOption = r.ReadUint32()
		}
		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			m.team = r.ReadInt8()
			m.effectItemId = r.ReadUint32()
			if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
				m.phase = r.ReadUint32()
			}
		} else {
			_ = r.ReadUint32()
		}
	}
}
