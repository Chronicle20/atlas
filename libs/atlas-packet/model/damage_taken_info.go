package model

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
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
	nX                byte
	bGuard            bool
	relativeDir       byte
	bPowerGuard       bool
	monsterId2        uint32
	powerGuard        bool
	hitX              int16
	hitY              int16
	characterX        int16
	characterY        int16
	expression        byte
}

func (m DamageTakenInfo) CharacterId() uint32         { return m.characterId }
func (m DamageTakenInfo) UpdateTime() uint32           { return m.updateTime }
func (m DamageTakenInfo) AttackIdx() DamageType        { return m.nAttackIdx }
func (m DamageTakenInfo) MagicElemAttr() DamageElementType { return m.nMagicElemAttr }
func (m DamageTakenInfo) Damage() int32                { return m.damage }
func (m DamageTakenInfo) ObstacleData() int16          { return m.obstacleData }
func (m DamageTakenInfo) MonsterTemplateId() uint32    { return m.monsterTemplateId }
func (m DamageTakenInfo) MonsterId() uint32            { return m.monsterId }
func (m DamageTakenInfo) Left() bool                   { return m.left }
func (m DamageTakenInfo) NX() byte                     { return m.nX }
func (m DamageTakenInfo) Guard() bool                  { return m.bGuard }
func (m DamageTakenInfo) RelativeDir() byte            { return m.relativeDir }
func (m DamageTakenInfo) PowerGuard() bool             { return m.bPowerGuard }
func (m DamageTakenInfo) MonsterId2() uint32           { return m.monsterId2 }
func (m DamageTakenInfo) PowerGuard2() bool            { return m.powerGuard }
func (m DamageTakenInfo) HitX() int16                  { return m.hitX }
func (m DamageTakenInfo) HitY() int16                  { return m.hitY }
func (m DamageTakenInfo) CharacterX() int16            { return m.characterX }
func (m DamageTakenInfo) CharacterY() int16            { return m.characterY }
func (m DamageTakenInfo) Expression() byte             { return m.expression }

func (m DamageTakenInfo) Operation() string {
	return CharacterDamageHandle
}

func (m DamageTakenInfo) String() string {
	return fmt.Sprintf("characterId [%d], updateTime [%d], nAttackIdx [%d], nMagicElemAttr [%d], damage [%d], obstacleData [%d], monsterTemplate [%d], monsterId [%d], left [%t], nX [%d], bGuard [%t], relativeDir [%d], bPowerGuard [%t], monsterId2 [%d], powerGuard [%t], hit [%d,%d], character [%d,%d], expression [%d]",
		m.characterId, m.updateTime, m.nAttackIdx, m.nMagicElemAttr, m.damage, m.obstacleData, m.monsterTemplateId, m.monsterId, m.left, m.nX, m.bGuard, m.relativeDir, m.bPowerGuard, m.monsterId2, m.powerGuard, m.hitX, m.hitY, m.characterX, m.characterY, m.expression)
}

func (m *DamageTakenInfo) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.nAttackIdx = DamageType(r.ReadInt8())
		m.nMagicElemAttr = DamageElementType(r.ReadInt8())
		m.damage = r.ReadInt32()

		if m.nAttackIdx == DamageTypePhysical || m.nAttackIdx == DamageTypeMagic {
			m.monsterTemplateId = r.ReadUint32()
			m.monsterId = r.ReadUint32()
			m.left = r.ReadBool()

			m.nX = r.ReadByte()
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				m.bGuard = r.ReadBool()
			}
			m.relativeDir = r.ReadByte()
			m.bPowerGuard = r.ReadBool()
			m.monsterId2 = r.ReadUint32()
			m.powerGuard = r.ReadBool()
			m.hitX = r.ReadInt16()
			m.hitY = r.ReadInt16()
			m.characterX = r.ReadInt16()
			m.characterY = r.ReadInt16()
		} else {
			m.obstacleData = r.ReadInt16()
		}
		m.expression = r.ReadByte()
	}
}

func (m DamageTakenInfo) Encode(_ logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(logrus.WithFields(logrus.Fields{}))
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt8(int8(m.nAttackIdx))
		w.WriteInt8(int8(m.nMagicElemAttr))
		w.WriteInt32(m.damage)

		if m.nAttackIdx == DamageTypePhysical || m.nAttackIdx == DamageTypeMagic {
			w.WriteInt(m.monsterTemplateId)
			w.WriteInt(m.monsterId)
			w.WriteBool(m.left)

			w.WriteByte(m.nX)
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				w.WriteBool(m.bGuard)
			}
			w.WriteByte(m.relativeDir)
			w.WriteBool(m.bPowerGuard)
			w.WriteInt(m.monsterId2)
			w.WriteBool(m.powerGuard)
			w.WriteInt16(m.hitX)
			w.WriteInt16(m.hitY)
			w.WriteInt16(m.characterX)
			w.WriteInt16(m.characterY)
		} else {
			w.WriteInt16(m.obstacleData)
		}
		w.WriteByte(m.expression)
		return w.Bytes()
	}
}
