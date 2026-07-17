package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// RPSGameWriter is the CRPSGameDlg::OnPacket mode-prefix dispatcher family.
// Four arms are implemented here (OPEN=8, START_SELECT=9, RESULT=11, END=13) —
// the modes atlas-rps emits to drive a game. START_SELECT (mode 9) is the
// round-start frame the client waits for after its serverbound START/CONTINUE
// sub-op: it enables the R/P/S buttons and arms the 30s selection timer
// (ida-rps-clientbound.md §0/§6). Without it the board's throw buttons never
// become clickable — the round cannot begin (live-confirmed 2026-07-17). Modes
// 6/7/10/12/14 also exist client-side but no atlas-rps path emits them today
// (mode 12 is a wire-distinct alias of 9), so they are intentionally NOT
// implemented here; adding writers for them would be unwired dead code.
//
// Every mode byte is IDENTICAL across all five versions (gms_v83/84/87/95,
// jms_v185) — unlike most dispatcher families, RPS_GAME has no per-version
// mode shift; only the RPS_GAME *opcode* shifts. See the IDA note §0/§6.
const RPSGameWriter = "RPSGame"

// Open — the OPEN arm (mode 8). Body: Decode4 int = the NPC template id.
// IDA: v83 0x7400ec, v84 0x761e10, v87 0x78acb0, v95 0x6d9e82, jms185 0x7ae4d7
// (docs/tasks/task-132-rps-npc-game/ida-rps-clientbound.md §1-§5). The client
// stores this int and uses it as an NPC template id to load Npc/{id:07d}.img
// for the dealer's face in the participation-fee confirm dialog
// (CUtilDlgEx::SetNPC → IWzResMan::GetObjectA); a value that is not a real NPC
// id in Npc.wz throws STG_E_FILENOTFOUND (0x80030002) and crashes the client.
// This is NOT the ante — the fee message is a static StringPool string with no
// amount; the fee is deducted server-side on confirm.
//
// packet-audit:fname CRPSGameDlg::OnPacket#OPEN
type Open struct {
	mode  byte
	npcId uint32
}

func NewRPSGameOpen(mode byte, npcId uint32) Open {
	return Open{mode: mode, npcId: npcId}
}

func (m Open) Mode() byte        { return m.mode }
func (m Open) NpcId() uint32     { return m.npcId }
func (m Open) Operation() string { return RPSGameWriter }
func (m Open) String() string {
	return fmt.Sprintf("rps game open [npcId=%d]", m.npcId)
}

func (m Open) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.npcId)
		return w.Bytes()
	}
}

func (m *Open) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.npcId = r.ReadUint32()
	}
}

// StartSelect — the START_SELECT arm (mode 9). No body — the mode byte alone.
// The client's dispatcher enables the three R/P/S buttons and arms the 30s
// selection timer on this frame (m_nNpcSelect=-1; m_tSwitchingTerm=120;
// m_tLimit=now+30000; enable m_pBtRPS[0..2]). IDA per-mode reads: v83 case 9 in
// sub_74024B (0x7402e9+), v84 sub_761F6F (0x76200d+), v87 sub_78AE23
// (0x78aec1+), v95 ProcessPacket case 9,12 (0x6d72ec+), jms185 sub_7AE636
// (0x7ae6d4+) — all "No read" (ida-rps-clientbound.md §1-§5). atlas-rps sends
// it in response to the serverbound START (sub-op 0) that opens the first round
// and the serverbound CONTINUE (sub-op 3) that opens each subsequent round; a
// tie re-enables the buttons client-side with no server frame, so no
// START_SELECT is sent on a tie (ida-rps-clientbound.md §16 / -serverbound.md §16).
//
// packet-audit:fname CRPSGameDlg::OnPacket#START_SELECT
type StartSelect struct {
	mode byte
}

func NewRPSGameStartSelect(mode byte) StartSelect {
	return StartSelect{mode: mode}
}

func (m StartSelect) Mode() byte        { return m.mode }
func (m StartSelect) Operation() string { return RPSGameWriter }
func (m StartSelect) String() string {
	return fmt.Sprintf("rps game start-select mode [%d]", m.mode)
}

func (m StartSelect) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte only; enable buttons + arm timer, no further reads
		return w.Bytes()
	}
}

func (m *StartSelect) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
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
