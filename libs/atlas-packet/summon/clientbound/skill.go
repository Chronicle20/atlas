package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const SummonSkillWriter = "SummonSkill"

// SummonSkill is the server -> client SUMMON_SKILL packet. The IDB-confirmed
// wire (CSummonedPool::OnHit@0x7a6e5a — dispatched on the LOWER of the swapped
// skill/damage opcodes; see summon-wire-truth.md) is just:
//
//	int  cid          // summon owner character id (consumed by dispatcher)
//	int  oid          // v95+ only (gated >= 95); v83/v87 have NO oid
//	byte (stance&0x7F) // a single stance byte
//
// There is NO summonSkillId int on the wire in ANY version — OnHit reads one
// Decode1, masks 0x7F, and replays the summon's skill animation. It drives the
// Beholder buff/heal aura visual and is broadcast map-wide (including the
// owner).
type SummonSkill struct {
	characterId uint32
	oid         uint32
	newStance   byte
}

func NewSummonSkill(characterId, oid uint32, newStance byte) SummonSkill {
	return SummonSkill{
		characterId: characterId,
		oid:         oid,
		newStance:   newStance,
	}
}

func (m SummonSkill) CharacterId() uint32 { return m.characterId }
func (m SummonSkill) Oid() uint32         { return m.oid }
func (m SummonSkill) NewStance() byte     { return m.newStance }
func (m SummonSkill) Operation() string   { return SummonSkillWriter }

func (m SummonSkill) String() string {
	return fmt.Sprintf("characterId [%d], oid [%d], newStance [%d]", m.characterId, m.oid, m.newStance)
}

func (m SummonSkill) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		// oid: present on ALL versions. cid is read upstream by
		// CUserPool::OnUserCommonPacket; CSummonedPool::OnPacket@0x938dd7 then does
		// one Decode4 = the oid before OnHit (the skill leaf). Wire = cid + oid +
		// stance (the old "no oid pre-95" reading missed the upstream cid read).
		w.WriteInt(m.oid)
		w.WriteByte(m.newStance)
		return w.Bytes()
	}
}

func (m *SummonSkill) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.oid = r.ReadUint32() // present on all versions (see Encode)
		m.newStance = r.ReadByte()
	}
}
