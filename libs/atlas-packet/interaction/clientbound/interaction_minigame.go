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
