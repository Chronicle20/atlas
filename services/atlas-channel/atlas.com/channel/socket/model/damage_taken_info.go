package model

import (
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
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

func NewDamageTakenInfo(characterId uint32) *DamageTakenInfo {
	return &DamageTakenInfo{characterId: characterId}
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

func (d *DamageTakenInfo) Decode(l logrus.FieldLogger, t tenant.Model, _ map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
		d.updateTime = r.ReadUint32()
		d.nAttackIdx = DamageType(r.ReadInt8())
		d.nMagicElemAttr = DamageElementType(r.ReadInt8())
		d.damage = r.ReadInt32()

		if d.nAttackIdx == DamageTypePhysical || d.nAttackIdx == DamageTypeMagic {
			d.monsterTemplateId = r.ReadUint32()
			d.monsterId = r.ReadUint32()
			d.left = r.ReadBool()

			d.nX = r.ReadByte()
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				d.bGuard = r.ReadBool()
			}
			d.relativeDir = r.ReadByte()
			d.bPowerGuard = r.ReadBool()
			d.monsterId2 = r.ReadUint32()
			d.powerGuard = r.ReadBool()
			d.hitX = r.ReadInt16()
			d.hitY = r.ReadInt16()
			d.characterX = r.ReadInt16()
			d.characterY = r.ReadInt16()
		} else {
			// not a monster, but an obstacle?
			d.obstacleData = r.ReadInt16()
		}
		d.expression = r.ReadByte()

		l.Debugf("Character [%d] has taken [%d] damage. updateTime [%d], nAttackIdx [%d], nMagicElemAttr [%d] obstacleData [%d]"+
			", monsterTemplate [%d], monsterId [%d], left [%t], nX [%d], bGuard [%t], relativeDir [%d], bPowerGuard [%t], monsterId2 [%d], "+
			"powerGuard [%t], hit x,y [%d,%d], character x,y [%d, %d], expression [%d].",
			d.characterId, d.damage, d.updateTime, d.nAttackIdx, d.nMagicElemAttr, d.obstacleData, d.monsterTemplateId, d.monsterId, d.left, d.nX,
			d.bGuard, d.relativeDir, d.bPowerGuard, d.monsterId2, d.powerGuard, d.hitX, d.hitY, d.characterX, d.characterY, d.expression)

	}
}

func (d *DamageTakenInfo) AttackIdx() DamageType {
	return d.nAttackIdx
}

func (d *DamageTakenInfo) Damage() int32 {
	return d.damage
}

func (d *DamageTakenInfo) MonsterTemplateId() uint32 {
	return d.monsterTemplateId
}

func (d *DamageTakenInfo) Left() bool {
	return d.left
}

func (d *DamageTakenInfo) PowerGuard() bool {
	return d.bPowerGuard
}

func (d *DamageTakenInfo) HitX() int16 {
	return d.hitX
}

func (d *DamageTakenInfo) HitY() int16 {
	return d.hitY
}
