package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MonsterMovementAckWriter = "MoveMonsterAck"

type MovementAck struct {
	uniqueId  uint32
	moveId    int16
	mp        uint16
	useSkills bool
	skillId   byte
	skillLevel byte
}

func NewMonsterMovementAck(uniqueId uint32, moveId int16, mp uint16, useSkills bool, skillId byte, skillLevel byte) MovementAck {
	return MovementAck{uniqueId: uniqueId, moveId: moveId, mp: mp, useSkills: useSkills, skillId: skillId, skillLevel: skillLevel}
}

func (m MovementAck) UniqueId() uint32   { return m.uniqueId }
func (m MovementAck) MoveId() int16      { return m.moveId }
func (m MovementAck) Mp() uint16         { return m.mp }
func (m MovementAck) UseSkills() bool    { return m.useSkills }
func (m MovementAck) SkillId() byte      { return m.skillId }
func (m MovementAck) SkillLevel() byte   { return m.skillLevel }
func (m MovementAck) Operation() string  { return MonsterMovementAckWriter }
func (m MovementAck) String() string {
	return fmt.Sprintf("uniqueId [%d], moveId [%d], useSkills [%t]", m.uniqueId, m.moveId, m.useSkills)
}

func (m MovementAck) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteInt16(m.moveId)
		w.WriteBool(m.useSkills)
		w.WriteShort(m.mp)
		w.WriteByte(m.skillId)
		w.WriteByte(m.skillLevel)
		return w.Bytes()
	}
}

func (m *MovementAck) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.moveId = r.ReadInt16()
		m.useSkills = r.ReadBool()
		m.mp = r.ReadUint16()
		m.skillId = r.ReadByte()
		m.skillLevel = r.ReadByte()
	}
}
