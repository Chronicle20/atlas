package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MonsterCarnivalHandle = "MonsterCarnival"

// MonsterCarnival is the serverbound MONSTER_CARNIVAL packet
// (CUIMonsterCarnival::RequestSend): the client requests a carnival action — it
// sends the current UI tab (request category) and the selected entry index.
//
// Byte layout (IDA-verified, identical across all 5 versions — Encode1 + Encode4):
//   - tab : byte  — Encode1(m_nCurTab); the carnival UI tab / request category
//   - idx : int32 — Encode4(m_dwCurIdx - 1); the selected entry index (the client
//     subtracts 1 from its 1-based current index before sending)
//
// IDA basis: CUIMonsterCarnival::RequestSend — v83 @0x8706d3 (op 0xDA), v84
// @0x89bdda (op 0xE0), v87 @0x8d93c3 (op 0xE7), v95 @0x80b4a0 (op 0x106), jms
// @0x903e24 (op 0xE5): `COutPacket(op); Encode1(m_nCurTab); Encode4(m_dwCurIdx-1);
// SendPacket`. The opcode shifts per version; the body shape is identical.
type MonsterCarnival struct {
	tab byte
	idx int32
}

func NewMonsterCarnival(tab byte, idx int32) MonsterCarnival {
	return MonsterCarnival{tab: tab, idx: idx}
}

func (m MonsterCarnival) Tab() byte         { return m.tab }
func (m MonsterCarnival) Idx() int32        { return m.idx }
func (m MonsterCarnival) Operation() string { return MonsterCarnivalHandle }
func (m MonsterCarnival) String() string {
	return fmt.Sprintf("tab [%d], idx [%d]", m.tab, m.idx)
}

func (m MonsterCarnival) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.tab)
		w.WriteInt32(m.idx)
		return w.Bytes()
	}
}

func (m *MonsterCarnival) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.tab = r.ReadByte()
		m.idx = r.ReadInt32()
	}
}
