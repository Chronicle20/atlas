package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MonsterCarnivalPartyCPWriter = "MonsterCarnivalPartyCP"

// MonsterCarnivalPartyCP is the clientbound MONSTER_CARNIVAL_PARTY_CP packet
// (CField_MonsterCarnival::OnTeamCP): the server updates a team's CP totals on
// the carnival scoreboard.
//
// Byte layout (IDA-verified, identical across all 5 versions — Decode1 + 2x Decode2):
//   - team  : byte   — Decode1; which team (0 or 1) -> SetTeamCP arg1
//   - cp    : uint16 — Decode2; SetTeamCP arg2 (current team CP)
//   - total : uint16 — Decode2; SetTeamCP arg3 (total team CP)
//
// IDA basis: CField_MonsterCarnival::OnTeamCP — v83 @0x56553e, v84 @0x572245,
// v87 @0x5902c4, v95 @0x55a2d0, jms @0x5b02f3: `v2=Decode1; v3=Decode2; v4=Decode2;
// SetTeamCP(v2, v3, v4)`. All five identical.
type MonsterCarnivalPartyCP struct {
	team  byte
	cp    uint16
	total uint16
}

func NewMonsterCarnivalPartyCP(team byte, cp uint16, total uint16) MonsterCarnivalPartyCP {
	return MonsterCarnivalPartyCP{team: team, cp: cp, total: total}
}

func (m MonsterCarnivalPartyCP) Team() byte        { return m.team }
func (m MonsterCarnivalPartyCP) Cp() uint16        { return m.cp }
func (m MonsterCarnivalPartyCP) Total() uint16     { return m.total }
func (m MonsterCarnivalPartyCP) Operation() string { return MonsterCarnivalPartyCPWriter }
func (m MonsterCarnivalPartyCP) String() string {
	return fmt.Sprintf("team [%d], cp [%d], total [%d]", m.team, m.cp, m.total)
}

func (m MonsterCarnivalPartyCP) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.team)
		w.WriteShort(m.cp)
		w.WriteShort(m.total)
		return w.Bytes()
	}
}

func (m *MonsterCarnivalPartyCP) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.team = r.ReadByte()
		m.cp = r.ReadUint16()
		m.total = r.ReadUint16()
	}
}
