package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MonsterCarnivalLeaveWriter = "MonsterCarnivalLeave"

// MonsterCarnivalLeave is the clientbound MONSTER_CARNIVAL_LEAVE packet
// (CField_MonsterCarnival::OnShowMemberOutMsg): when a participant quits the
// carnival the server announces it; the client builds a chat-log line, choosing a
// "leader quit / new leader appointed" variant when the leader flag is set.
//
// Byte layout (IDA-verified, identical across all 5 versions — 2x Decode1 + DecodeStr):
//   - leader : byte   — Decode1; leader flag (==6 => "leader quit, X appointed" variant)
//   - team   : byte   — Decode1; team color selector (!=0 => MAPLE_BLUE, 0 => MAPLE_RED)
//   - name   : string — DecodeStr; the quitting character's name (length-prefixed ascii)
//
// IDA basis: CField_MonsterCarnival::OnShowMemberOutMsg — v83 @0x565962, v84 @0x572669,
// v87 @0x5906e3, v95 @0x55ad80, jms @0x5b070f: `v2=(Decode1()==6); if(Decode1())
// MAPLE_BLUE else MAPLE_RED; DecodeStr(name); <leader or normal quit msg>`.
type MonsterCarnivalLeave struct {
	leader byte
	team   byte
	name   string
}

func NewMonsterCarnivalLeave(leader byte, team byte, name string) MonsterCarnivalLeave {
	return MonsterCarnivalLeave{leader: leader, team: team, name: name}
}

func (m MonsterCarnivalLeave) Leader() byte      { return m.leader }
func (m MonsterCarnivalLeave) Team() byte        { return m.team }
func (m MonsterCarnivalLeave) Name() string      { return m.name }
func (m MonsterCarnivalLeave) Operation() string { return MonsterCarnivalLeaveWriter }
func (m MonsterCarnivalLeave) String() string {
	return fmt.Sprintf("leader [%d], team [%d], name [%s]", m.leader, m.team, m.name)
}

func (m MonsterCarnivalLeave) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.leader)
		w.WriteByte(m.team)
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *MonsterCarnivalLeave) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.leader = r.ReadByte()
		m.team = r.ReadByte()
		m.name = r.ReadAsciiString()
	}
}
