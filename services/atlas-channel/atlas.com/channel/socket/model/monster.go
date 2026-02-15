package model

import (
	"atlas-channel/tool"
	"errors"
	"math"
	"sort"
	"time"

	"github.com/Chronicle20/atlas-constants/monster"
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

func (m *MonsterBurnedInfo) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt(m.characterId)
		w.WriteInt(m.skillId)
		w.WriteInt(m.damage)
		w.WriteInt(m.interval)
		w.WriteInt(m.end)
		w.WriteInt(m.dotCount)
	}
}

type MonsterTemporaryStat struct {
	burnedInfo []MonsterBurnedInfo
	stats      map[monster.TemporaryStatType]MonsterTemporaryStatValue
}

func (m *MonsterTemporaryStat) EncodeMask(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		mask := tool.Uint128{}
		for _, v := range m.stats {
			mask = mask.Or(v.StatType().Mask())
		}

		w.WriteInt(uint32(mask.H >> 32))
		w.WriteInt(uint32(mask.H & 0xFFFFFFFF))
		w.WriteInt(uint32(mask.L >> 32))
		w.WriteInt(uint32(mask.L & 0xFFFFFFFF))
	}
}

func (m *MonsterTemporaryStat) Encode(l logrus.FieldLogger, tenant tenant.Model, ops map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		m.EncodeMask(l, tenant, ops)(w)

		keys := make([]MonsterTemporaryStatType, 0)
		for _, v := range m.stats {
			keys = append(keys, v.statType)
		}

		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Shift() < keys[j].Shift()
		})

		// Create a slice of values sorted by the keys' index
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
					b.Encode(l, tenant, ops)(w)
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
	}
}

func monsterStatExpiry(_ time.Time) int16 {
	// The client field is int16 which cannot hold a meaningful absolute time.
	// Return -1 (permanent display) and let the server manage expiry via
	// STATUS_EXPIRED / STATUS_CANCELLED events.
	return -1
}

func NewMonsterTemporaryStat() *MonsterTemporaryStat {
	return &MonsterTemporaryStat{
		stats: make(map[monster.TemporaryStatType]MonsterTemporaryStatValue),
	}
}

func (m *Monster) SetTemporaryStat(stat *MonsterTemporaryStat) {
	m.monsterTemporaryStat = *stat
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

type Monster struct {
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

func NewMonster(x int16, y int16, stance byte, fh int16, appearType MonsterAppearType, team int8) Monster {
	return Monster{
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

func (m *Monster) Encode(l logrus.FieldLogger, tenant tenant.Model, ops map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 12) || tenant.Region() == "JMS" {
			m.monsterTemporaryStat.Encode(l, tenant, ops)(w)
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
		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 12) || tenant.Region() == "JMS" {
			w.WriteInt8(m.team)
			w.WriteInt(m.effectItemId)
			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 83) || tenant.Region() == "JMS" {
				w.WriteInt(m.phase)
			}
		} else {
			// TODO proper temp stat encoding for GMS v12
			w.WriteInt(0)
		}
	}
}
