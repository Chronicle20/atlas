package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// InteractionMiniGameEnter - a visitor joined a game room. This is the GAME
// shape of the shared ENTER arm (mode 4): where the shop-room ENTER
// (InteractionEnter + interaction.NewBaseVisitor) stops after the visitor
// name, the game dialogs read a trailing 20-byte win/tie/loss record.
//
// IDA-derived read order (same two-stage dispatch as the room-enter blob):
//   - CMiniRoomBaseDlg::OnEnterBase (v95 0x638f80, typed): byte slot,
//     AvatarLook blob (CMiniRoomBaseDlg::DecodeAvatar), string name, then
//     `m_anJobCode[slot] = Decode2()` — the same per-avatar uint16 jobCode the
//     room-enter avatar list carries, version-gated identically
//     (enterHasJobCode: (GMS && MajorAtLeast(84)) || JMS; absent in v83 —
//     ida-notes.md §G5 jobCode grounding table), then virtual OnEnter.
//   - COmokDlg::OnEnter (v95 0x6812e0) / CMemoryGameDlg::OnEnter (v95
//     0x628980): GW_MiniGameRecord::Decode (0x4f2ad0 = DecodeBuffer(20) =
//     5 x int32: Unknown, Wins, Ties, Losses, Points). v83:
//     COmokDlg::OnEnter = sub_6E3BCC @ 0x6e3bcc, record read via sub_4E42FC
//     — structurally identical ("%s HAVE ENTERED" chat + minigame sound).
//
// Sent to the room owner when ENTERED fires; the joining visitor gets the
// full InteractionMiniGameRoom snapshot instead (design §5).
//
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#EnterMiniGame
type InteractionMiniGameEnter struct {
	mode   byte
	player MiniGameRoomPlayer
}

func NewInteractionMiniGameEnter(mode byte, player MiniGameRoomPlayer) InteractionMiniGameEnter {
	return InteractionMiniGameEnter{mode: mode, player: player}
}

func (m InteractionMiniGameEnter) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameEnter) String() string {
	return fmt.Sprintf("minigame enter slot [%d] name [%s]", m.player.Slot, m.player.Name)
}

func (m InteractionMiniGameEnter) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.player.Slot)
		w.WriteByteArray(m.player.Avatar.Encode(l, ctx)(options))
		w.WriteAsciiString(m.player.Name)
		if enterHasJobCode(ctx) {
			w.WriteShort(m.player.JobCode)
		}
		w.WriteInt(m.player.Record.Unknown)
		w.WriteInt(m.player.Record.Wins)
		w.WriteInt(m.player.Record.Ties)
		w.WriteInt(m.player.Record.Losses)
		w.WriteInt(m.player.Record.Points)
		return w.Bytes()
	}
}

func (m *InteractionMiniGameEnter) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.player.Slot = r.ReadByte()
		m.player.Avatar.Decode(l, ctx)(r, options)
		m.player.Name = r.ReadAsciiString()
		if enterHasJobCode(ctx) {
			m.player.JobCode = r.ReadUint16()
		}
		m.player.Record = interaction.GameRecord{
			Unknown: r.ReadUint32(),
			Wins:    r.ReadUint32(),
			Ties:    r.ReadUint32(),
			Losses:  r.ReadUint32(),
			Points:  r.ReadUint32(),
		}
	}
}
