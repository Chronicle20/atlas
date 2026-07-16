package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// InteractionMiniGameReady - visitor readied up
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#MemoryGameReady
type InteractionMiniGameReady struct{ mode byte }

func NewInteractionMiniGameReady(mode byte) InteractionMiniGameReady {
	return InteractionMiniGameReady{mode: mode}
}
func (m InteractionMiniGameReady) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameReady) String() string    { return "minigame ready" }
func (m InteractionMiniGameReady) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}
func (m *InteractionMiniGameReady) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) { m.mode = r.ReadByte() }
}

// InteractionMiniGameUnready - visitor cancelled ready
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#MemoryGameUnready
type InteractionMiniGameUnready struct{ mode byte }

func NewInteractionMiniGameUnready(mode byte) InteractionMiniGameUnready {
	return InteractionMiniGameUnready{mode: mode}
}
func (m InteractionMiniGameUnready) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameUnready) String() string    { return "minigame unready" }
func (m InteractionMiniGameUnready) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}
func (m *InteractionMiniGameUnready) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) { m.mode = r.ReadByte() }
}

// InteractionMiniGameRequestTie - a player asked for a tie
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#MemoryGameRequestTie
type InteractionMiniGameRequestTie struct{ mode byte }

func NewInteractionMiniGameRequestTie(mode byte) InteractionMiniGameRequestTie {
	return InteractionMiniGameRequestTie{mode: mode}
}
func (m InteractionMiniGameRequestTie) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameRequestTie) String() string    { return "minigame request tie" }
func (m InteractionMiniGameRequestTie) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}
func (m *InteractionMiniGameRequestTie) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) { m.mode = r.ReadByte() }
}

// InteractionMiniGameAnswerTie - a tie request was denied (accept path emits
// RESULT instead of this arm; see body func doc below).
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#MemoryGameAnswerTie
type InteractionMiniGameAnswerTie struct{ mode byte }

func NewInteractionMiniGameAnswerTie(mode byte) InteractionMiniGameAnswerTie {
	return InteractionMiniGameAnswerTie{mode: mode}
}
func (m InteractionMiniGameAnswerTie) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameAnswerTie) String() string    { return "minigame answer tie" }
func (m InteractionMiniGameAnswerTie) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}
func (m *InteractionMiniGameAnswerTie) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) { m.mode = r.ReadByte() }
}

// InteractionMiniGameRetreatRequest - a player asked to retreat (undo).
// Bodyless — same shape as the serverbound send (ida-notes.md §G2,
// COmokDlg::OnRetreatRequest v83 @ 0x6e416b). No Cosmic reference exists;
// §G2 is the sole layout authority, verified on gms_v83 and gms_v95.
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#MemoryGameRetreatRequest
type InteractionMiniGameRetreatRequest struct{ mode byte }

func NewInteractionMiniGameRetreatRequest(mode byte) InteractionMiniGameRetreatRequest {
	return InteractionMiniGameRetreatRequest{mode: mode}
}
func (m InteractionMiniGameRetreatRequest) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameRetreatRequest) String() string    { return "minigame retreat request" }
func (m InteractionMiniGameRetreatRequest) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}
func (m *InteractionMiniGameRetreatRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) { m.mode = r.ReadByte() }
}

// InteractionMiniGameRetreatAnswer - retreat request answered. On accept the
// client pops stoneCount stones from the tail of the move history and honors
// turnSlot verbatim (my-turn = turnSlot == mySlot); on decline only the
// accept discriminator is written. ida-notes.md §G2 is the sole layout
// authority (no Cosmic reference), verified on gms_v83
// (COmokDlg::OnRetreatResult @ 0x6e41f9) and gms_v95 (@ 0x684620). N and
// turnSlot are server-chosen (Task 15); the wire supports any values.
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#MemoryGameRetreatAnswer
type InteractionMiniGameRetreatAnswer struct {
	mode       byte
	accept     bool
	stoneCount byte
	turnSlot   byte
}

func NewInteractionMiniGameRetreatAnswer(mode byte, accept bool, stoneCount byte, turnSlot byte) InteractionMiniGameRetreatAnswer {
	return InteractionMiniGameRetreatAnswer{mode: mode, accept: accept, stoneCount: stoneCount, turnSlot: turnSlot}
}
func (m InteractionMiniGameRetreatAnswer) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameRetreatAnswer) String() string {
	return fmt.Sprintf("minigame retreat answer accept [%t] stoneCount [%d] turnSlot [%d]", m.accept, m.stoneCount, m.turnSlot)
}
func (m InteractionMiniGameRetreatAnswer) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		accept := byte(0)
		if m.accept {
			accept = 1
		}
		w.WriteByte(accept)
		if m.accept {
			w.WriteByte(m.stoneCount)
			w.WriteByte(m.turnSlot)
		}
		return w.Bytes()
	}
}
func (m *InteractionMiniGameRetreatAnswer) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.accept = r.ReadByte() == 1
		if m.accept {
			m.stoneCount = r.ReadByte()
			m.turnSlot = r.ReadByte()
		}
	}
}

// InteractionMiniGameSkip - a player's turn timed out / was skipped
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#MemoryGameSkip
type InteractionMiniGameSkip struct {
	mode byte
	who  byte // slot whose turn it now is (the next mover), NOT the skipper —
	// COmokDlg::OnTimeOver (v83 0x6e472e) sets my-turn = (who == mySlot); see
	// ida-notes.md §G5 SKIP for the reconciliation with Cosmic's owner/visitor writers.
}

func NewInteractionMiniGameSkip(mode byte, who byte) InteractionMiniGameSkip {
	return InteractionMiniGameSkip{mode: mode, who: who}
}
func (m InteractionMiniGameSkip) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameSkip) String() string    { return fmt.Sprintf("minigame skip who [%d]", m.who) }
func (m InteractionMiniGameSkip) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.who)
		return w.Bytes()
	}
}
func (m *InteractionMiniGameSkip) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.who = r.ReadByte()
	}
}

// InteractionMiniGamePutStoneError - Omok invalid-move rejection (Omok-only;
// MemoryGame has no stone placement). COmokDlg::OnPutStoneCheckerErr reads a
// single errorCode byte and shows a red chat line: the version-specific
// "double 3s" (renju double-three forbidden) code -> "You have double 3s",
// otherwise -> "You can't put it there". The double-3 code is version-specific
// (v48 60 / v61 61 / v72 61 / v79 66 / v83..v95 67 / jms 64), so a producer that
// emits a specific error type must config-resolve the code per version. IDA:
// v48 sub_573A10 @0x573a10, v61 sub_5F7B5F @0x5f7b5f, v72 sub_64E84D @0x64e84d,
// v79 sub_672622 @0x672622, v83 COmokDlg::OnPutStoneCheckerErr @0x6e4065,
// v87 @0x721b74, v95 @0x680360, jms @0x72b593.
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#MemoryGamePutStoneError
type InteractionMiniGamePutStoneError struct {
	mode      byte
	errorCode byte
}

func NewInteractionMiniGamePutStoneError(mode byte, errorCode byte) InteractionMiniGamePutStoneError {
	return InteractionMiniGamePutStoneError{mode: mode, errorCode: errorCode}
}
func (m InteractionMiniGamePutStoneError) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGamePutStoneError) String() string {
	return fmt.Sprintf("minigame put-stone error code [%d]", m.errorCode)
}
func (m InteractionMiniGamePutStoneError) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.errorCode)
		return w.Bytes()
	}
}
func (m *InteractionMiniGamePutStoneError) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadByte()
	}
}

// InteractionMiniGameStartOmok - Omok game started. `firstMover` is the raw
// wire byte per ida-notes.md §G1 (COmokDlg::OnUserStart): the client grants
// the first move to the slot that is NOT equal to this byte.
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#MemoryGameStartOmok
type InteractionMiniGameStartOmok struct {
	mode       byte
	firstMover byte
}

func NewInteractionMiniGameStartOmok(mode byte, firstMover byte) InteractionMiniGameStartOmok {
	return InteractionMiniGameStartOmok{mode: mode, firstMover: firstMover}
}
func (m InteractionMiniGameStartOmok) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameStartOmok) String() string {
	return fmt.Sprintf("minigame start omok firstMover [%d]", m.firstMover)
}
func (m InteractionMiniGameStartOmok) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.firstMover)
		return w.Bytes()
	}
}
func (m *InteractionMiniGameStartOmok) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.firstMover = r.ReadByte()
	}
}

// InteractionMiniGameStartMatchCards - Match Cards game started. Same
// first-mover byte semantics as Omok (ida-notes.md §G1), plus the shuffled
// deck: byte count followed by count little-endian int32 card ids
// (CMemoryGameDlg::OnUserStart).
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#MemoryGameStartMatchCards
type InteractionMiniGameStartMatchCards struct {
	mode       byte
	firstMover byte
	deck       []uint32
}

func NewInteractionMiniGameStartMatchCards(mode byte, firstMover byte, deck []uint32) InteractionMiniGameStartMatchCards {
	return InteractionMiniGameStartMatchCards{mode: mode, firstMover: firstMover, deck: deck}
}
func (m InteractionMiniGameStartMatchCards) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameStartMatchCards) String() string {
	return fmt.Sprintf("minigame start match cards firstMover [%d] deck %v", m.firstMover, m.deck)
}
func (m InteractionMiniGameStartMatchCards) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.firstMover)
		w.WriteByte(byte(len(m.deck)))
		for _, cardId := range m.deck {
			w.WriteInt(cardId)
		}
		return w.Bytes()
	}
}
func (m *InteractionMiniGameStartMatchCards) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.firstMover = r.ReadByte()
		count := r.ReadByte()
		m.deck = make([]uint32, count)
		for i := range m.deck {
			m.deck[i] = r.ReadUint32()
		}
	}
}

// InteractionMiniGameMoveStone - an Omok stone was placed. `stoneType` is the
// placing player's color (1/2) per ida-notes.md §G5 MOVE_STONE
// (COmokDlg::OnPutStoneChecker).
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#MemoryGameMoveStone
type InteractionMiniGameMoveStone struct {
	mode      byte
	x         uint32
	y         uint32
	stoneType byte
}

func NewInteractionMiniGameMoveStone(mode byte, x uint32, y uint32, stoneType byte) InteractionMiniGameMoveStone {
	return InteractionMiniGameMoveStone{mode: mode, x: x, y: y, stoneType: stoneType}
}
func (m InteractionMiniGameMoveStone) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameMoveStone) String() string {
	return fmt.Sprintf("minigame move stone x [%d] y [%d] type [%d]", m.x, m.y, m.stoneType)
}
func (m InteractionMiniGameMoveStone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.x)
		w.WriteInt(m.y)
		w.WriteByte(m.stoneType)
		return w.Bytes()
	}
}
func (m *InteractionMiniGameMoveStone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.x = r.ReadUint32()
		m.y = r.ReadUint32()
		m.stoneType = r.ReadByte()
	}
}

// InteractionMiniGameCardSelectFirst - first card flip of a Match Cards turn.
// The turn byte is always 1 for the first flip (ida-notes.md §G5
// SELECT_CARD/FLIP_CARD, CMemoryGameDlg::OnTurnUpCard) and is forwarded to
// the opponent only, per design §3.2.
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#MemoryGameCardSelectFirst
type InteractionMiniGameCardSelectFirst struct {
	mode byte
	slot byte
}

func NewInteractionMiniGameCardSelectFirst(mode byte, slot byte) InteractionMiniGameCardSelectFirst {
	return InteractionMiniGameCardSelectFirst{mode: mode, slot: slot}
}
func (m InteractionMiniGameCardSelectFirst) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameCardSelectFirst) String() string {
	return fmt.Sprintf("minigame card select first slot [%d]", m.slot)
}
func (m InteractionMiniGameCardSelectFirst) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(1)
		w.WriteByte(m.slot)
		return w.Bytes()
	}
}
func (m *InteractionMiniGameCardSelectFirst) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		_ = r.ReadByte() // turn byte, always 1 for the first flip
		m.slot = r.ReadByte()
	}
}

// InteractionMiniGameCardSelectSecond - second card flip of a Match Cards
// turn. The turn byte is always 0 for the second flip (ida-notes.md §G5
// SELECT_CARD/FLIP_CARD, CMemoryGameDlg::OnTurnUpCard) and is forwarded to
// both players, per design §3.2. resultType: 0 owner-mismatch, 1
// visitor-mismatch, 2 owner-match, 3 visitor-match.
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#MemoryGameCardSelectSecond
type InteractionMiniGameCardSelectSecond struct {
	mode       byte
	slot       byte
	firstSlot  byte
	resultType byte
}

func NewInteractionMiniGameCardSelectSecond(mode byte, slot byte, firstSlot byte, resultType byte) InteractionMiniGameCardSelectSecond {
	return InteractionMiniGameCardSelectSecond{mode: mode, slot: slot, firstSlot: firstSlot, resultType: resultType}
}
func (m InteractionMiniGameCardSelectSecond) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameCardSelectSecond) String() string {
	return fmt.Sprintf("minigame card select second slot [%d] firstSlot [%d] resultType [%d]", m.slot, m.firstSlot, m.resultType)
}
func (m InteractionMiniGameCardSelectSecond) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(0)
		w.WriteByte(m.slot)
		w.WriteByte(m.firstSlot)
		w.WriteByte(m.resultType)
		return w.Bytes()
	}
}
func (m *InteractionMiniGameCardSelectSecond) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		_ = r.ReadByte() // turn byte, always 0 for the second flip
		m.slot = r.ReadByte()
		m.firstSlot = r.ReadByte()
		m.resultType = r.ReadByte()
	}
}

// InteractionMiniGameResult - game concluded. resultType selects the shape
// (0 win, 1 tie, 2 forfeit-win); the winnerSlot byte (carried here as
// visitorWon) is present for win/forfeit only and is OMITTED for a tie —
// per ida-notes.md §G5 RESULT (COmokDlg::OnGameResult v83 @ 0x6e4463 /
// CMemoryGameDlg::OnGameResult v83 @ 0x64e423, byte-identical shape):
//
//	byte resultType              # 1 = tie; else a winnerSlot byte follows
//	if resultType != 1: byte winnerSlot
//	<20-byte record>             # owner  (5 x int32: Unknown, Wins, Ties, Losses, Points)
//	<20-byte record>             # visitor
//
// NOTE: this is a correction of the plan's Task 4 draft layout, which added
// a bool written on every shape plus int32/int16 padding blocks and a
// trailing tie byte not supported by the IDA read order — see plan.md
// Task 4 and the task-4 commit body for the reconciliation.
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#MemoryGameResult
type InteractionMiniGameResult struct {
	mode          byte
	resultType    byte
	visitorWon    bool
	ownerRecord   interaction.GameRecord
	visitorRecord interaction.GameRecord
}

func NewInteractionMiniGameResult(mode byte, resultType byte, visitorWon bool, ownerRecord interaction.GameRecord, visitorRecord interaction.GameRecord) InteractionMiniGameResult {
	return InteractionMiniGameResult{mode: mode, resultType: resultType, visitorWon: visitorWon, ownerRecord: ownerRecord, visitorRecord: visitorRecord}
}
func (m InteractionMiniGameResult) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameResult) String() string {
	return fmt.Sprintf("minigame result resultType [%d] visitorWon [%t]", m.resultType, m.visitorWon)
}
func (m InteractionMiniGameResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.resultType)
		if m.resultType != 1 {
			winnerSlot := byte(0)
			if m.visitorWon {
				winnerSlot = 1
			}
			w.WriteByte(winnerSlot)
		}
		writeMiniGameRecord(w, m.ownerRecord)
		writeMiniGameRecord(w, m.visitorRecord)
		return w.Bytes()
	}
}
func (m *InteractionMiniGameResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.resultType = r.ReadByte()
		if m.resultType != 1 {
			m.visitorWon = r.ReadByte() == 1
		}
		m.ownerRecord = readMiniGameRecord(r)
		m.visitorRecord = readMiniGameRecord(r)
	}
}

func writeMiniGameRecord(w *response.Writer, rec interaction.GameRecord) {
	w.WriteInt(rec.Unknown)
	w.WriteInt(rec.Wins)
	w.WriteInt(rec.Ties)
	w.WriteInt(rec.Losses)
	w.WriteInt(rec.Points)
}

func readMiniGameRecord(r *request.Reader) interaction.GameRecord {
	return interaction.GameRecord{
		Unknown: r.ReadUint32(),
		Wins:    r.ReadUint32(),
		Ties:    r.ReadUint32(),
		Losses:  r.ReadUint32(),
		Points:  r.ReadUint32(),
	}
}
