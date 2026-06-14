package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MonsterCarnivalStartWriter = "MonsterCarnivalStart"

// MonsterCarnivalStart is the clientbound MONSTER_CARNIVAL_START packet
// (CField_MonsterCarnival::OnEnter): when a character enters a Monster Carnival
// field the server pushes the initial scoreboard state — the player's team, the
// personal and per-team CP totals, and the list of currently-summonable beings
// (one "spelled" flag byte per summon slot).
//
// Byte layout (IDA-verified, identical across all 5 versions):
//   - team          : byte   — Decode1; the entering player's team (0 or 1) -> SetTeam
//   - personalCp    : uint16 — Decode2; SetPersonalCP arg1 (current personal CP)
//   - personalTotal : uint16 — Decode2; SetPersonalCP arg2 (total personal CP)
//   - myTeamCp      : uint16 — Decode2; SetTeamCP(team, ...) current CP
//   - myTeamTotal   : uint16 — Decode2; SetTeamCP(team, ...) total CP
//   - enemyTeamCp   : uint16 — Decode2; SetTeamCP(!team, ...) current CP
//   - enemyTeamTotal: uint16 — Decode2; SetTeamCP(!team, ...) total CP
//   - spelled[]     : []byte — Decode1 looped once per summon slot (m_aSummonedMob);
//     each byte is the summon's "spelled" level (>0 => InsertSpelledData).
//     The loop count is the client's local m_aSummonedMob array size, so
//     the slice length is the agreed-upon slot count (caller-supplied).
//
// IDA basis: CField_MonsterCarnival::OnEnter — v83 @0x565397, v84 @0x57209e,
// v87 @0x59011d, v95 @0x55a6c0, jms @0x5b014c: Decode1 team, then 6x Decode2
// (SetPersonalCP + 2x SetTeamCP), then `for (i=0; i < m_aSummonedMob.size; i++)
// Decode1()` filling InsertSpelledData. All five share this shape.
type MonsterCarnivalStart struct {
	team           byte
	personalCp     uint16
	personalTotal  uint16
	myTeamCp       uint16
	myTeamTotal    uint16
	enemyTeamCp    uint16
	enemyTeamTotal uint16
	spelled        []byte
}

func NewMonsterCarnivalStart(team byte, personalCp uint16, personalTotal uint16, myTeamCp uint16, myTeamTotal uint16, enemyTeamCp uint16, enemyTeamTotal uint16, spelled []byte) MonsterCarnivalStart {
	return MonsterCarnivalStart{
		team:           team,
		personalCp:     personalCp,
		personalTotal:  personalTotal,
		myTeamCp:       myTeamCp,
		myTeamTotal:    myTeamTotal,
		enemyTeamCp:    enemyTeamCp,
		enemyTeamTotal: enemyTeamTotal,
		spelled:        spelled,
	}
}

func (m MonsterCarnivalStart) Team() byte             { return m.team }
func (m MonsterCarnivalStart) PersonalCp() uint16     { return m.personalCp }
func (m MonsterCarnivalStart) PersonalTotal() uint16  { return m.personalTotal }
func (m MonsterCarnivalStart) MyTeamCp() uint16       { return m.myTeamCp }
func (m MonsterCarnivalStart) MyTeamTotal() uint16    { return m.myTeamTotal }
func (m MonsterCarnivalStart) EnemyTeamCp() uint16    { return m.enemyTeamCp }
func (m MonsterCarnivalStart) EnemyTeamTotal() uint16 { return m.enemyTeamTotal }
func (m MonsterCarnivalStart) Spelled() []byte        { return m.spelled }
func (m MonsterCarnivalStart) Operation() string      { return MonsterCarnivalStartWriter }
func (m MonsterCarnivalStart) String() string {
	return fmt.Sprintf("team [%d], personalCp [%d], personalTotal [%d], myTeamCp [%d], myTeamTotal [%d], enemyTeamCp [%d], enemyTeamTotal [%d], spelled %v",
		m.team, m.personalCp, m.personalTotal, m.myTeamCp, m.myTeamTotal, m.enemyTeamCp, m.enemyTeamTotal, m.spelled)
}

func (m MonsterCarnivalStart) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.team)
		w.WriteShort(m.personalCp)
		w.WriteShort(m.personalTotal)
		w.WriteShort(m.myTeamCp)
		w.WriteShort(m.myTeamTotal)
		w.WriteShort(m.enemyTeamCp)
		w.WriteShort(m.enemyTeamTotal)
		for _, s := range m.spelled {
			w.WriteByte(s)
		}
		return w.Bytes()
	}
}

func (m *MonsterCarnivalStart) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.team = r.ReadByte()
		m.personalCp = r.ReadUint16()
		m.personalTotal = r.ReadUint16()
		m.myTeamCp = r.ReadUint16()
		m.myTeamTotal = r.ReadUint16()
		m.enemyTeamCp = r.ReadUint16()
		m.enemyTeamTotal = r.ReadUint16()
		// The summon-slot count is not length-prefixed on the wire — the client
		// loops to its local m_aSummonedMob size. For round-trip symmetry the
		// decoder consumes exactly len(m.spelled) bytes (the agreed slot count).
		n := len(m.spelled)
		out := make([]byte, n)
		for i := 0; i < n; i++ {
			out[i] = r.ReadByte()
		}
		m.spelled = out
	}
}
