package clientbound

import (
	"context"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterSkillChangeWriter = "CharacterSkillChange"

type CharacterSkillChange struct {
	exclRequestSent bool
	skillId         uint32
	level           uint32
	masterLevel     uint32
	expiration      int64
	sn              bool
}

func NewCharacterSkillChange(exclRequestSent bool, skillId uint32, level byte, masterLevel byte, expiration time.Time, sn bool) CharacterSkillChange {
	return CharacterSkillChange{
		exclRequestSent: exclRequestSent,
		skillId:         skillId,
		level:           uint32(level),
		masterLevel:     uint32(masterLevel),
		expiration:      skillMsTime(expiration),
		sn:              sn,
	}
}

func (m CharacterSkillChange) ExclRequestSent() bool { return m.exclRequestSent }
func (m CharacterSkillChange) SkillId() uint32        { return m.skillId }
func (m CharacterSkillChange) Level() uint32           { return m.level }
func (m CharacterSkillChange) MasterLevel() uint32     { return m.masterLevel }
func (m CharacterSkillChange) Sn() bool                { return m.sn }
func (m CharacterSkillChange) Operation() string       { return CharacterSkillChangeWriter }
func (m CharacterSkillChange) String() string {
	return fmt.Sprintf("skillId [%d], level [%d]", m.skillId, m.level)
}

func skillMsTime(t time.Time) int64 {
	if t.IsZero() {
		return -1
	}
	return t.Unix()*int64(10000000) + int64(116444736000000000)
}

func (m CharacterSkillChange) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.exclRequestSent)
		w.WriteShort(1) // number of skills
		w.WriteInt(m.skillId)
		w.WriteInt(m.level)
		w.WriteInt(m.masterLevel)
		w.WriteInt64(m.expiration)
		w.WriteBool(m.sn)
		return w.Bytes()
	}
}

func (m *CharacterSkillChange) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.exclRequestSent = r.ReadBool()
		_ = r.ReadUint16() // count
		m.skillId = r.ReadUint32()
		m.level = r.ReadUint32()
		m.masterLevel = r.ReadUint32()
		m.expiration = r.ReadInt64()
		m.sn = r.ReadBool()
	}
}
