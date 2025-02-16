package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterDamageHandle = "CharacterDamageHandle"

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

func CharacterDamageHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTime := r.ReadUint32()
		nAttackIdx := DamageType(r.ReadInt8())
		nMagicElemAttr := DamageElementType(r.ReadInt8())
		damage := r.ReadUint32()
		obstacleData := int16(0)
		monsterTemplateId := uint32(0)
		monsterId := uint32(0)
		direction := byte(0)
		nX := byte(0)
		bGuard := false
		relativeDir := byte(0)
		bPowerGuard := false
		monsterId2 := uint32(0)
		powerGuard := false
		hitX := int16(0)
		hitY := int16(0)
		characterX := int16(0)
		characterY := int16(0)
		if nAttackIdx == DamageTypeObstacle {
			// not a monster, but an obstacle?
			obstacleData = r.ReadInt16()
		} else if nAttackIdx == DamageTypePhysical || nAttackIdx == DamageTypeMagic {
			monsterTemplateId = r.ReadUint32()
			monsterId = r.ReadUint32()
			direction = r.ReadByte()

			nX = r.ReadByte()
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				bGuard = r.ReadBool()
			}
			relativeDir = r.ReadByte()
			bPowerGuard = r.ReadBool()
			monsterId2 = r.ReadUint32()
			powerGuard = r.ReadBool()
			hitX = r.ReadInt16()
			hitY = r.ReadInt16()
			characterX = r.ReadInt16()
			characterY = r.ReadInt16()
		}
		expression := r.ReadByte()
		l.Debugf("Character [%d] has taken [%d] damage. updateTime [%d], nAttackIdx [%d], nMagicElemAttr [%d] obstacleData [%d]"+
			", monsterTemplate [%d], monsterId [%d], direction [%d], nX [%d], bGuard [%t], relativeDir [%d], bPowerGuard [%t], monsterId2 [%d], "+
			"powerGuard [%t], hit x,y [%d,%d], character x,y [%d, %d], expression [%d].",
			s.CharacterId(), damage, updateTime, nAttackIdx, nMagicElemAttr, obstacleData, monsterTemplateId, monsterId, direction, nX,
			bGuard, relativeDir, bPowerGuard, monsterId2, powerGuard, hitX, hitY, characterX, characterY, expression)

		// TODO process mana reflection
		// TODO process achilles
		// TODO process combo barrier
		// TODO process Body Pressure
		// TODO process PowerGuard
		// TODO process Paladin Divine Shield
		// TODO process Aran High Defense
		// TODO process MagicGuard
		// TODO process MesoGuard
		// TODO decrease battleship hp
	}
}
