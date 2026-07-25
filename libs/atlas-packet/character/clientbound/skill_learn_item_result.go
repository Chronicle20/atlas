package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CharacterSkillLearnItemResultWriter = "CharacterSkillLearnItemResult"

// SkillLearnItemResult - CWvsContext::OnSkillLearnItemResult
// packet-audit:fname CWvsContext::OnSkillLearnItemResult
//
// Wire layout (IDB-verified all 10 versions):
//
//	Decode1 bOnExclRequest — v84+ ONLY (leading byte); server sends 1 so the
//	                         requesting/local client clears its exclusive-request
//	                         lock. Observers ignore it.
//	Decode4 characterId    — resolved via CUserPool::GetUser; glow renders on that avatar for any observer
//	Decode1 isMasteryBook
//	Decode4 skillId        — decoded-then-discarded by the client (every version)
//	Decode4 masterLevel    — decoded-then-discarded by the client (every version)
//	Decode1 canUse         — gates the on-avatar glow; effect renders only when 1
//	Decode1 success        — success vs failure sound/message (local user only)
//
// Atlas sends the real skillId/masterLevel even though the client discards them
// (task-125 design §D-4 — do not copy Cosmic's hardcoded zeros).
//
// VERSION GATE (IDB-verified, NOT speculative): the leading bOnExclRequest byte
// is present iff MajorVersion() >= 84. 15-byte body: gms_48/61/72/79/83.
// 16-byte body: gms_84/87/92/95/jms_185. This is a real v84≠v83 exception — v84
// serverbound is byte-identical to v83 but its clientbound diverges. Do NOT gate
// at >=87. bOnExclRequest is NOT domain data (server always sends 1), so it is
// not a struct field: Encode writes it, Decode consumes-and-discards it.
type SkillLearnItemResult struct {
	characterId   uint32
	isMasteryBook bool
	skillId       uint32
	masterLevel   uint32
	canUse        bool
	success       bool
}

// skillLearnResultHasExclByte reports whether this tenant's client reads the
// leading bOnExclRequest byte (GMS v84+; JMS v185 satisfies >=84 naturally).
func skillLearnResultHasExclByte(t tenant.Model) bool {
	return t.MajorVersion() >= 84
}

func NewSkillLearnItemResult(characterId uint32, isMasteryBook bool, skillId uint32, masterLevel uint32, canUse bool, success bool) SkillLearnItemResult {
	return SkillLearnItemResult{
		characterId:   characterId,
		isMasteryBook: isMasteryBook,
		skillId:       skillId,
		masterLevel:   masterLevel,
		canUse:        canUse,
		success:       success,
	}
}

func (m SkillLearnItemResult) CharacterId() uint32 { return m.characterId }
func (m SkillLearnItemResult) IsMasteryBook() bool { return m.isMasteryBook }
func (m SkillLearnItemResult) SkillId() uint32     { return m.skillId }
func (m SkillLearnItemResult) MasterLevel() uint32 { return m.masterLevel }
func (m SkillLearnItemResult) CanUse() bool        { return m.canUse }
func (m SkillLearnItemResult) Success() bool       { return m.success }
func (m SkillLearnItemResult) Operation() string   { return CharacterSkillLearnItemResultWriter }

func (m SkillLearnItemResult) String() string {
	return fmt.Sprintf("skill learn item result characterId [%d] mastery [%t] skillId [%d] masterLevel [%d] canUse [%t] success [%t]",
		m.characterId, m.isMasteryBook, m.skillId, m.masterLevel, m.canUse, m.success)
}

func (m SkillLearnItemResult) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if skillLearnResultHasExclByte(t) {
			w.WriteBool(true) // bOnExclRequest — clears the requester's exclusive-request lock (v84+)
		}
		w.WriteInt(m.characterId)
		w.WriteBool(m.isMasteryBook)
		w.WriteInt(m.skillId)
		w.WriteInt(m.masterLevel)
		w.WriteBool(m.canUse)
		w.WriteBool(m.success)
		return w.Bytes()
	}
}

func (m *SkillLearnItemResult) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if skillLearnResultHasExclByte(t) {
			_ = r.ReadBool() // bOnExclRequest — consumed then discarded (not domain data)
		}
		m.characterId = r.ReadUint32()
		m.isMasteryBook = r.ReadBool()
		m.skillId = r.ReadUint32()
		m.masterLevel = r.ReadUint32()
		m.canUse = r.ReadBool()
		m.success = r.ReadBool()
	}
}
