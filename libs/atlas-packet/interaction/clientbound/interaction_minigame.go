package clientbound

import (
	"context"
	"fmt"

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
