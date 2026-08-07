package model

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type DamageType int8

type DamageElementType int8

const (
	DamageTypeMagic    = DamageType(0)
	DamageTypePhysical = DamageType(-1)
	DamageTypeCounter  = DamageType(-2)
	DamageTypeObstacle = DamageType(-3)
	DamageTypeStat     = DamageType(-4)

	DamageElementTypeNone      = DamageElementType(0)
	DamageElementTypeIce       = DamageElementType(1)
	DamageElementTypeFire      = DamageElementType(2)
	DamageElementTypeLightning = DamageElementType(3)
)

const CharacterDamageHandle = "CharacterDamageHandle"

func NewDamageTakenInfo(characterId uint32) DamageTakenInfo {
	return DamageTakenInfo{characterId: characterId}
}

type DamageTakenInfo struct {
	characterId       uint32
	updateTime        uint32
	nAttackIdx        DamageType
	nMagicElemAttr    DamageElementType
	damage            int32
	obstacleData      int16
	monsterTemplateId uint32
	monsterId         uint32
	left              bool
	reflect           byte
	guard             bool
	blockByte         byte
	// hasReflectExtension mirrors the client's variable-length tail: the
	// 14-byte reflect extension is written iff the client set bKnockback
	// or a non-zero reflect echo (CUserLocal::SetDamaged, verified v83/
	// v87/v95/jms185).
	hasReflectExtension bool
	isPowerGuard        bool
	reflectTargetMobId  uint32
	hitAction           byte
	hitX                int16
	hitY                int16
	characterX          int16
	characterY          int16
	stanceFlags         byte
}

func (m DamageTakenInfo) CharacterId() uint32              { return m.characterId }
func (m DamageTakenInfo) UpdateTime() uint32               { return m.updateTime }
func (m DamageTakenInfo) AttackIdx() DamageType            { return m.nAttackIdx }
func (m DamageTakenInfo) MagicElemAttr() DamageElementType { return m.nMagicElemAttr }
func (m DamageTakenInfo) Damage() int32                    { return m.damage }
func (m DamageTakenInfo) ObstacleData() int16              { return m.obstacleData }
func (m DamageTakenInfo) MonsterTemplateId() uint32        { return m.monsterTemplateId }
func (m DamageTakenInfo) MonsterId() uint32                { return m.monsterId }
func (m DamageTakenInfo) Left() bool                       { return m.left }
func (m DamageTakenInfo) Reflect() byte                    { return m.reflect }
func (m DamageTakenInfo) Guard() bool                      { return m.guard }
func (m DamageTakenInfo) BlockByte() byte                  { return m.blockByte }
func (m DamageTakenInfo) HasReflectExtension() bool        { return m.hasReflectExtension }
func (m DamageTakenInfo) IsPowerGuard() bool               { return m.isPowerGuard }
func (m DamageTakenInfo) ReflectTargetMobId() uint32       { return m.reflectTargetMobId }
func (m DamageTakenInfo) HitAction() byte                  { return m.hitAction }
func (m DamageTakenInfo) HitX() int16                      { return m.hitX }
func (m DamageTakenInfo) HitY() int16                      { return m.hitY }
func (m DamageTakenInfo) CharacterX() int16                { return m.characterX }
func (m DamageTakenInfo) CharacterY() int16                { return m.characterY }
func (m DamageTakenInfo) StanceFlags() byte                { return m.stanceFlags }

func (m DamageTakenInfo) Operation() string {
	return CharacterDamageHandle
}

func (m DamageTakenInfo) String() string {
	return fmt.Sprintf("characterId [%d], updateTime [%d], nAttackIdx [%d], nMagicElemAttr [%d], damage [%d], obstacleData [%d], monsterTemplate [%d], monsterId [%d], left [%t], reflect [%d], guard [%t], blockByte [%d], hasReflectExtension [%t], isPowerGuard [%t], reflectTargetMobId [%d], hitAction [%d], hit [%d,%d], character [%d,%d], stanceFlags [%d]",
		m.characterId, m.updateTime, m.nAttackIdx, m.nMagicElemAttr, m.damage, m.obstacleData, m.monsterTemplateId, m.monsterId, m.left, m.reflect, m.guard, m.blockByte, m.hasReflectExtension, m.isPowerGuard, m.reflectTargetMobId, m.hitAction, m.hitX, m.hitY, m.characterX, m.characterY, m.stanceFlags)
}

// Legacy layout gates (design §2a, verified per-version IDBs):
//
//	preV61Layout  — gms_v48 only: NO nMagicElemAttr byte; the reflect
//	                extension is 10 bytes (no charX/charY).
//	preV83NonMob  — gms_v48/v61/v72/v79: the non-mob (obstacle/stat)
//	                branch has NO trailing stanceFlags byte.
//
// v61 through v92 decode as v83 (mob-hit byte-identical). The mob branch's
// trailing stanceFlags is present on every version. v95-GMS adds the bGuard
// byte (gated >=95); jms takes the no-bGuard branch.
func gmsBelow(t tenant.Model, major uint16) bool {
	return t.Region() == "GMS" && t.MajorVersion() < major
}

func (m *DamageTakenInfo) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	preV61Layout := gmsBelow(t, 61)
	preV83NonMob := gmsBelow(t, 83)
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.nAttackIdx = DamageType(r.ReadInt8())
		if !preV61Layout {
			m.nMagicElemAttr = DamageElementType(r.ReadInt8())
		}
		m.damage = r.ReadInt32()

		if m.nAttackIdx >= DamageTypePhysical {
			m.monsterTemplateId = r.ReadUint32()
			m.monsterId = r.ReadUint32()
			m.left = r.ReadBool()

			m.reflect = r.ReadByte()
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				m.guard = r.ReadBool()
			}
			m.blockByte = r.ReadByte()
			// The client writes the reflect extension iff it set bKnockback
			// or a reflect echo. Neither flag is fully recoverable from
			// earlier bytes, so presence is derived from the remaining
			// length: without the extension exactly the 1-byte stance
			// remains; with it, 11 bytes (v48, 10-byte ext) or 15 bytes
			// (v61+, 14-byte ext) remain — so Available() > 1 detects it on
			// every version.
			if r.Available() > 1 {
				m.hasReflectExtension = true
				m.isPowerGuard = r.ReadBool()
				m.reflectTargetMobId = r.ReadUint32()
				m.hitAction = r.ReadByte()
				m.hitX = r.ReadInt16()
				m.hitY = r.ReadInt16()
				if !preV61Layout {
					m.characterX = r.ReadInt16()
					m.characterY = r.ReadInt16()
				}
			}
			m.stanceFlags = r.ReadByte()
		} else {
			m.obstacleData = r.ReadInt16()
			if !preV83NonMob {
				m.stanceFlags = r.ReadByte()
			}
		}
	}
}

func (m DamageTakenInfo) Encode(_ logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(logrus.WithFields(logrus.Fields{}))
	t := tenant.MustFromContext(ctx)
	preV61Layout := gmsBelow(t, 61)
	preV83NonMob := gmsBelow(t, 83)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt8(int8(m.nAttackIdx))
		if !preV61Layout {
			w.WriteInt8(int8(m.nMagicElemAttr))
		}
		w.WriteInt32(m.damage)

		if m.nAttackIdx >= DamageTypePhysical {
			w.WriteInt(m.monsterTemplateId)
			w.WriteInt(m.monsterId)
			w.WriteBool(m.left)

			w.WriteByte(m.reflect)
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				w.WriteBool(m.guard)
			}
			w.WriteByte(m.blockByte)
			if m.hasReflectExtension {
				w.WriteBool(m.isPowerGuard)
				w.WriteInt(m.reflectTargetMobId)
				w.WriteByte(m.hitAction)
				w.WriteInt16(m.hitX)
				w.WriteInt16(m.hitY)
				if !preV61Layout {
					w.WriteInt16(m.characterX)
					w.WriteInt16(m.characterY)
				}
			}
			w.WriteByte(m.stanceFlags)
		} else {
			w.WriteInt16(m.obstacleData)
			if !preV83NonMob {
				w.WriteByte(m.stanceFlags)
			}
		}
		return w.Bytes()
	}
}
