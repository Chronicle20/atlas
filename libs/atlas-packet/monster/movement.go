package monster

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const MonsterMovementHandle = "MonsterMovementHandle"

type Movement struct {
	uniqueId              uint32
	moveId                int16
	dwFlag                byte
	nActionAndDir         int8
	skillData             uint32
	multiTargetForBall    model.MultiTargetForBall
	randTimeForAreaAttack model.RandTimeForAreaAttack
	moveFlags             byte
	hackedCode            uint32
	flyCtxTargetX         uint32
	flyCtxTargetY         uint32
	hackedCodeCRC         uint32
	movement              model.Movement
	bChasing              byte
	hasTarget             byte
	bChasing2             byte
	bChasingHack          byte
	tChaseDuration        uint32
}

func (m Movement) UniqueId() uint32                             { return m.uniqueId }
func (m Movement) MoveId() int16                                { return m.moveId }
func (m Movement) DwFlag() byte                                 { return m.dwFlag }
func (m Movement) ActionAndDir() int8                           { return m.nActionAndDir }
func (m Movement) SkillData() uint32                            { return m.skillData }
func (m Movement) SkillId() int16                               { return int16(m.skillData & 0xFF) }
func (m Movement) SkillLevel() int16                            { return int16(m.skillData >> 8 & 0xFF) }
func (m Movement) MonsterMoveStartResult() bool                 { return m.dwFlag > 0 }
func (m Movement) MultiTargetForBall() model.MultiTargetForBall { return m.multiTargetForBall }
func (m Movement) RandTimeForAreaAttack() model.RandTimeForAreaAttack {
	return m.randTimeForAreaAttack
}
func (m Movement) MovementData() model.Movement { return m.movement }

func (m Movement) Operation() string {
	return MonsterMovementHandle
}

func (m Movement) String() string {
	return fmt.Sprintf("uniqueId [%d] moveId [%d] dwFlag [%d] nActionAndDir [%d] skillData [%d] elements [%d]",
		m.uniqueId, m.moveId, m.dwFlag, m.nActionAndDir, m.skillData, len(m.movement.Elements))
}

func (m Movement) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteInt16(m.moveId)
		w.WriteByte(m.dwFlag)
		w.WriteInt8(m.nActionAndDir)
		w.WriteInt(m.skillData)

		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			w.WriteByteArray(m.multiTargetForBall.Encode(l, ctx)(options))
			w.WriteByteArray(m.randTimeForAreaAttack.Encode(l, ctx)(options))
		}

		w.WriteByte(m.moveFlags)
		w.WriteInt(m.hackedCode)
		w.WriteInt(m.flyCtxTargetX)
		w.WriteInt(m.flyCtxTargetY)
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			w.WriteInt(m.hackedCodeCRC)
		}

		w.WriteByteArray(m.movement.Encode(l, ctx)(options))

		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			w.WriteByte(m.bChasing)
			w.WriteByte(m.hasTarget)
			w.WriteByte(m.bChasing2)
			w.WriteByte(m.bChasingHack)
			w.WriteInt(m.tChaseDuration)
		}
		return w.Bytes()
	}
}

func (m *Movement) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.moveId = r.ReadInt16()
		m.dwFlag = r.ReadByte()
		m.nActionAndDir = r.ReadInt8()
		m.skillData = r.ReadUint32()

		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			m.multiTargetForBall.Decode(l, ctx)(r, options)
			m.randTimeForAreaAttack.Decode(l, ctx)(r, options)
		}

		m.moveFlags = r.ReadByte()
		m.hackedCode = r.ReadUint32()
		m.flyCtxTargetX = r.ReadUint32()
		m.flyCtxTargetY = r.ReadUint32()
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			m.hackedCodeCRC = r.ReadUint32()
		}

		m.movement.Decode(l, ctx)(r, options)

		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			m.bChasing = r.ReadByte()
			m.hasTarget = r.ReadByte()
			m.bChasing2 = r.ReadByte()
			m.bChasingHack = r.ReadByte()
			m.tChaseDuration = r.ReadUint32()
		}
	}
}
