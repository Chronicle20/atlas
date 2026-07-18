package clientbound

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// skillChangeHasExpiration reports whether the per-skill 8-byte expiration field
// is on the wire for this tenant. The field is read by the client via
// DecodeBuffer(8) in CWvsContext::OnChangeSkillRecordResult. It was introduced at
// GMS v83: legacy GMS clients (<83) read only skillId/level/masterLevel (3 ints)
// per skill and then the trailing sn byte — no expiration. IDA-verified:
// v79 @0x968f0e reads 3 Decode4 per skill (no DecodeBuffer); v83 @0xa1e48c reads
// 3 Decode4 + DecodeBuffer(v10, 8). JMS keeps the field.
func skillChangeHasExpiration(ctx context.Context) bool {
	t := tenant.MustFromContext(ctx)
	return t.Region() != "GMS" || t.MajorVersion() >= 83
}

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
func (m CharacterSkillChange) SkillId() uint32       { return m.skillId }
func (m CharacterSkillChange) Level() uint32         { return m.level }
func (m CharacterSkillChange) MasterLevel() uint32   { return m.masterLevel }
func (m CharacterSkillChange) Sn() bool              { return m.sn }
func (m CharacterSkillChange) Operation() string     { return CharacterSkillChangeWriter }
func (m CharacterSkillChange) String() string {
	return fmt.Sprintf("skillId [%d], level [%d]", m.skillId, m.level)
}

func skillMsTime(t time.Time) int64 {
	if t.IsZero() {
		return -1
	}
	return t.Unix()*int64(10000000) + int64(116444736000000000)
}

func (m CharacterSkillChange) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.exclRequestSent)
		w.WriteShort(1) // number of skills
		w.WriteInt(m.skillId)
		w.WriteInt(m.level)
		w.WriteInt(m.masterLevel)
		if skillChangeHasExpiration(ctx) {
			w.WriteInt64(m.expiration)
		}
		w.WriteBool(m.sn)
		return w.Bytes()
	}
}

func (m *CharacterSkillChange) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.exclRequestSent = r.ReadBool()
		_ = r.ReadUint16() // count
		m.skillId = r.ReadUint32()
		m.level = r.ReadUint32()
		m.masterLevel = r.ReadUint32()
		if skillChangeHasExpiration(ctx) {
			m.expiration = r.ReadInt64()
		}
		m.sn = r.ReadBool()
	}
}
