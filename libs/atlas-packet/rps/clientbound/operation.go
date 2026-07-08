package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// RPSGameWriter is the CRPSGameDlg::OnPacket mode-prefix dispatcher family.
// Only three arms are implemented here (OPEN=8, RESULT=11, END=13) — the ONLY
// modes atlas-rps emits. Modes 6/7/9/10/12/14 exist client-side (see
// docs/tasks/task-132-rps-npc-game/ida-rps-clientbound.md §0/§6) but are never
// sent by the server; adding writers for them would be unwired dead code, so
// they are intentionally NOT implemented here.
//
// Every mode byte is IDENTICAL across all five versions (gms_v83/84/87/95,
// jms_v185) — unlike most dispatcher families, RPS_GAME has no per-version
// mode shift; only the RPS_GAME *opcode* shifts. See the IDA note §0/§6.
const RPSGameWriter = "RPSGame"

// Open — the OPEN arm (mode 8). Body: Decode4 int (ante / participation fee).
// IDA: v83 0x7400ec, v84 0x761e10, v87 0x78acb0, v95 0x6d9e82, jms185 0x7ae4d7
// (docs/tasks/task-132-rps-npc-game/ida-rps-clientbound.md §1-§5). The
// StringPool notice string that follows in the client is a static resource,
// NOT a packet field.
//
// packet-audit:fname CRPSGameDlg::OnPacket#OPEN
type Open struct {
	mode byte
	ante uint32
}

func NewRPSGameOpen(mode byte, ante uint32) Open {
	return Open{mode: mode, ante: ante}
}

func (m Open) Mode() byte        { return m.mode }
func (m Open) Ante() uint32      { return m.ante }
func (m Open) Operation() string { return RPSGameWriter }
func (m Open) String() string {
	return fmt.Sprintf("rps game open [ante=%d]", m.ante)
}

func (m Open) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.ante)
		return w.Bytes()
	}
}

func (m *Open) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.ante = r.ReadUint32()
	}
}

// Result — the RESULT arm (mode 11). Body: Decode1 npcThrow (byte) then
// Decode1 straightVictoryCount, read as a SIGNED __int8 by the client (it
// branches `if (v < 0)`). IDA reads: v83 0x740298/0x7402a3, v84
// 0x761fbc/0x761fc7, v87 0x78ae70/0x78ae7b, v95 0x6d7372/0x6d737d, jms185
// 0x7ae683/0x7ae68e (ida-rps-clientbound.md §1-§5). v95 field names confirm
// the semantics: npcThrow = m_nNpcSelect, straightVictoryCount =
// m_nCntStraightVictories.
//
// packet-audit:fname CRPSGameDlg::OnPacket#RESULT
type Result struct {
	mode                 byte
	npcThrow             byte
	straightVictoryCount int8
}

func NewRPSGameResult(mode byte, npcThrow byte, straightVictoryCount int8) Result {
	return Result{mode: mode, npcThrow: npcThrow, straightVictoryCount: straightVictoryCount}
}

func (m Result) Mode() byte                 { return m.mode }
func (m Result) NpcThrow() byte             { return m.npcThrow }
func (m Result) StraightVictoryCount() int8 { return m.straightVictoryCount }
func (m Result) Operation() string          { return RPSGameWriter }
func (m Result) String() string {
	return fmt.Sprintf("rps game result [npcThrow=%d, straightVictoryCount=%d]", m.npcThrow, m.straightVictoryCount)
}

func (m Result) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.npcThrow)
		w.WriteInt8(m.straightVictoryCount)
		return w.Bytes()
	}
}

func (m *Result) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.npcThrow = r.ReadByte()
		m.straightVictoryCount = r.ReadInt8()
	}
}

// End — the CLOSE arm (mode 13). No body — `CWnd::Destroy` with no further
// wire reads. IDA: v83 0x74009e, v84 0x761dc2, v87 0x78ac5a, v95 0x6d9ff0,
// jms185 0x7ae489 (ida-rps-clientbound.md §1-§5).
//
// packet-audit:fname CRPSGameDlg::OnPacket#END
type End struct {
	mode byte
}

func NewRPSGameEnd(mode byte) End {
	return End{mode: mode}
}

func (m End) Mode() byte        { return m.mode }
func (m End) Operation() string { return RPSGameWriter }
func (m End) String() string {
	return fmt.Sprintf("rps game end mode [%d]", m.mode)
}

func (m End) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte only; CWnd::Destroy, no further reads
		return w.Bytes()
	}
}

func (m *End) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
