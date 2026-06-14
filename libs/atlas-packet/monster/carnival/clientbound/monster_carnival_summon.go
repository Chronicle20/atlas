package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MonsterCarnivalSummonWriter = "MonsterCarnivalSummon"

// MonsterCarnivalSummon is the clientbound MONSTER_CARNIVAL_SUMMON packet — the
// SUMMON mode of CField_MonsterCarnival::OnRequestResult (dispatched when the
// dispatcher's sub-op arg != 0). The server confirms a successful summon/spell
// request and tells the client which slot was summoned by whom.
//
// Byte layout (IDA-verified, identical across all 5 versions — 2x Decode1 + DecodeStr):
//   - tab  : byte   — Decode1; RequestResult arg1 (the requesting tab / category)
//   - idx  : byte   — Decode1; RequestResult arg2 (the summoned slot index)
//   - name : string — DecodeStr; the requesting character's name (length-prefixed ascii)
//
// IDA basis: CField_MonsterCarnival::OnRequestResult — v83 @0x56557d, v84 @0x572284,
// v87 @0x590303, v95 @0x55a890, jms @0x5b0332. The `bResult != 0` (SUMMON) branch:
// `v3=Decode1; v4=Decode1; DecodeStr(name); RequestResult(v3, v4, name)`. This is a
// DISTINCT wire shape from the MESSAGE mode (single Decode1) of the same dispatcher.
type MonsterCarnivalSummon struct {
	tab  byte
	idx  byte
	name string
}

func NewMonsterCarnivalSummon(tab byte, idx byte, name string) MonsterCarnivalSummon {
	return MonsterCarnivalSummon{tab: tab, idx: idx, name: name}
}

func (m MonsterCarnivalSummon) Tab() byte         { return m.tab }
func (m MonsterCarnivalSummon) Idx() byte         { return m.idx }
func (m MonsterCarnivalSummon) Name() string      { return m.name }
func (m MonsterCarnivalSummon) Operation() string { return MonsterCarnivalSummonWriter }
func (m MonsterCarnivalSummon) String() string {
	return fmt.Sprintf("tab [%d], idx [%d], name [%s]", m.tab, m.idx, m.name)
}

func (m MonsterCarnivalSummon) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.tab)
		w.WriteByte(m.idx)
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *MonsterCarnivalSummon) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.tab = r.ReadByte()
		m.idx = r.ReadByte()
		m.name = r.ReadAsciiString()
	}
}
