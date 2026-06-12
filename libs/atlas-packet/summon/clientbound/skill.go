package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const SummonSkillWriter = "SummonSkill"

// SummonSkill is the server -> client SUMMON_SKILL packet, encoded per Cosmic
// PacketCreator.summonSkill (PacketCreator.java:4569): int cid, int
// summonSkillId, byte newStance. It drives the Beholder buff/heal aura visual:
// the client plays the summon's skill animation at the given stance for the
// owning character's summon. Broadcast map-wide (the aura visual shows for
// everyone, including the owner).
type SummonSkill struct {
	characterId   uint32
	summonSkillId uint32
	newStance     byte
}

func NewSummonSkill(characterId, summonSkillId uint32, newStance byte) SummonSkill {
	return SummonSkill{
		characterId:   characterId,
		summonSkillId: summonSkillId,
		newStance:     newStance,
	}
}

func (m SummonSkill) CharacterId() uint32   { return m.characterId }
func (m SummonSkill) SummonSkillId() uint32 { return m.summonSkillId }
func (m SummonSkill) NewStance() byte       { return m.newStance }
func (m SummonSkill) Operation() string     { return SummonSkillWriter }

func (m SummonSkill) String() string {
	return fmt.Sprintf("characterId [%d], summonSkillId [%d], newStance [%d]", m.characterId, m.summonSkillId, m.newStance)
}

func (m SummonSkill) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	_ = tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteInt(m.summonSkillId)
		w.WriteByte(m.newStance)
		return w.Bytes()
	}
}

func (m *SummonSkill) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	_ = tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.summonSkillId = r.ReadUint32()
		m.newStance = r.ReadByte()
	}
}
