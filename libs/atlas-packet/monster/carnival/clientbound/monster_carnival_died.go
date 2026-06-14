package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MonsterCarnivalDiedWriter = "MonsterCarnivalDied"

// MonsterCarnivalDied is the clientbound MONSTER_CARNIVAL_DIED packet
// (CField_MonsterCarnival::OnProcessForDeath): when a participant is defeated the
// server announces it; the client composes a chat-log line ("X has become unable
// to fight...") from the team color, the character name, and the lost CP.
//
// Byte layout (IDA-verified, identical across all 5 versions — Decode1 + DecodeStr + Decode1):
//   - team   : byte   — Decode1; team color selector (!=0 => MAPLE_BLUE, 0 => MAPLE_RED)
//   - name   : string — DecodeStr; the defeated character's name (length-prefixed ascii)
//   - lostCp : byte   — Decode1; CP lost by the team (0 => "no CP lost" message variant)
//
// IDA basis: CField_MonsterCarnival::OnProcessForDeath — v83 @0x5657e7, v84 @0x5724ee,
// v87 @0x590568, v95 @0x55ab90, jms @0x5b0597: `v2=Decode1; DecodeStr(name);
// v3=Decode1; if(v2) MAPLE_BLUE else MAPLE_RED; if(v3<=0) <no-cp msg> else <lost-cp msg>`.
type MonsterCarnivalDied struct {
	team   byte
	name   string
	lostCp byte
}

func NewMonsterCarnivalDied(team byte, name string, lostCp byte) MonsterCarnivalDied {
	return MonsterCarnivalDied{team: team, name: name, lostCp: lostCp}
}

func (m MonsterCarnivalDied) Team() byte        { return m.team }
func (m MonsterCarnivalDied) Name() string      { return m.name }
func (m MonsterCarnivalDied) LostCp() byte      { return m.lostCp }
func (m MonsterCarnivalDied) Operation() string { return MonsterCarnivalDiedWriter }
func (m MonsterCarnivalDied) String() string {
	return fmt.Sprintf("team [%d], name [%s], lostCp [%d]", m.team, m.name, m.lostCp)
}

func (m MonsterCarnivalDied) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.team)
		w.WriteAsciiString(m.name)
		w.WriteByte(m.lostCp)
		return w.Bytes()
	}
}

func (m *MonsterCarnivalDied) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.team = r.ReadByte()
		m.name = r.ReadAsciiString()
		m.lostCp = r.ReadByte()
	}
}
