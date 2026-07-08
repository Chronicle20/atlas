package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// MiniGameRoomPlayer is one occupant of an Omok / Match Cards room as it
// appears in the room-enter snapshot. Its avatar+name flow through the avatar
// list; its 20-byte win/tie/loss/points record flows through the SEPARATE
// record list (see InteractionMiniGameRoom for the two-list layout).
type MiniGameRoomPlayer struct {
	Slot   byte
	Avatar model.Avatar
	Name   string
	Record interaction.GameRecord
}

// InteractionMiniGameRoom is the EnterResultSuccess body for an Omok /
// Match Cards game room — the full room snapshot the server sends when a
// player creates or joins a game (the game analogue of the shop-room enter
// that InteractionEnterResultSuccess composes via interaction.Room).
//
// Wire layout (IDA-derived, gms_v83; ida-notes.md §G5 "Room-enter blob — FULL
// RESOLUTION"). The blob is assembled by three client functions:
//   - CMiniRoomBaseDlg::OnEnterResultStatic (v83 0x65dff3): byte roomType
//     (nonzero ⇒ success ⇒ MiniRoomFactory builds the dialog).
//   - CMiniRoomBaseDlg::OnEnterResultBase (v83 0x65ec3d): byte capacity
//     (m_nMaxUsers), byte yourSlot (m_nMyPosition), then a 0xFF-terminated
//     AVATAR list — each entry {byte slot, AvatarLook blob, string name}.
//     The vtable+92 `IsEntrusted()` predicate is 0 for both game dialogs
//     (sub_48315F `return 0`), so the owner-slot-0 Decode4/RegisterEmployer
//     int32 branch is DEAD for games: every occupant, owner included, is a
//     full avatar. (v95 adds a per-avatar Decode2 jobCode here; v83 does not.)
//   - COmokDlg::OnEnterResult (v83 0x6e388e) / CMemoryGameDlg::OnEnterResult
//     (v83 0x64db..): a SECOND 0xFF-terminated RECORD list — each entry
//     {byte slot, 20-byte record = 5×int32 via sub_4E42FC} — then string
//     title, byte gameKind, byte tournament, and (tournament only) byte round.
//
// Cross-checked against Cosmic getMiniGame (PacketCreator.java:4653-4688),
// which addCharLook()s the owner (confirming no int32 branch) and writes the
// two lists exactly this way.
//
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#EnterResultSuccessMiniGame
type InteractionMiniGameRoom struct {
	mode       byte
	roomType   interaction.RoomType
	capacity   byte
	yourSlot   byte
	players    []MiniGameRoomPlayer
	title      string
	gameKind   byte
	tournament bool
	round      byte
}

// NewInteractionMiniGameRoom builds a game room-enter blob. mode is the ROOM /
// EnterResult dispatcher mode (5, the same the shop-room enter uses); roomType
// is interaction.OmokRoomType (1) or interaction.MatchCardRoomType (2);
// capacity is m_nMaxUsers (2 for both games); yourSlot is the recipient's slot
// (0 owner / 1 visitor). round is written only when tournament is true.
func NewInteractionMiniGameRoom(mode byte, roomType interaction.RoomType, capacity byte, yourSlot byte, players []MiniGameRoomPlayer, title string, gameKind byte, tournament bool, round byte) InteractionMiniGameRoom {
	return InteractionMiniGameRoom{
		mode:       mode,
		roomType:   roomType,
		capacity:   capacity,
		yourSlot:   yourSlot,
		players:    players,
		title:      title,
		gameKind:   gameKind,
		tournament: tournament,
		round:      round,
	}
}

func (m InteractionMiniGameRoom) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameRoom) String() string {
	return fmt.Sprintf("minigame room enter roomType [%d] yourSlot [%d] players [%d]", m.roomType, m.yourSlot, len(m.players))
}

func (m InteractionMiniGameRoom) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(byte(m.roomType))
		w.WriteByte(m.capacity)
		w.WriteByte(m.yourSlot)
		// Avatar list (0xFF-terminated).
		for _, p := range m.players {
			w.WriteByte(p.Slot)
			w.WriteByteArray(p.Avatar.Encode(l, ctx)(options))
			w.WriteAsciiString(p.Name)
		}
		w.WriteByte(0xFF)
		// Record list (0xFF-terminated), separate from the avatar list.
		for _, p := range m.players {
			w.WriteByte(p.Slot)
			w.WriteInt(p.Record.Unknown)
			w.WriteInt(p.Record.Wins)
			w.WriteInt(p.Record.Ties)
			w.WriteInt(p.Record.Losses)
			w.WriteInt(p.Record.Points)
		}
		w.WriteByte(0xFF)
		w.WriteAsciiString(m.title)
		w.WriteByte(m.gameKind)
		w.WriteBool(m.tournament)
		if m.tournament {
			w.WriteByte(m.round)
		}
		return w.Bytes()
	}
}

func (m *InteractionMiniGameRoom) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.roomType = interaction.RoomType(r.ReadByte())
		m.capacity = r.ReadByte()
		m.yourSlot = r.ReadByte()

		// Avatar list.
		type avatarEntry struct {
			slot   byte
			avatar model.Avatar
			name   string
		}
		var avatars []avatarEntry
		for {
			slot := r.ReadByte()
			if slot == 0xFF {
				break
			}
			var a model.Avatar
			a.Decode(l, ctx)(r, options)
			avatars = append(avatars, avatarEntry{slot: slot, avatar: a, name: r.ReadAsciiString()})
		}

		// Record list, keyed by slot back onto the avatar entries.
		records := map[byte]interaction.GameRecord{}
		for {
			slot := r.ReadByte()
			if slot == 0xFF {
				break
			}
			records[slot] = interaction.GameRecord{
				Unknown: r.ReadUint32(),
				Wins:    r.ReadUint32(),
				Ties:    r.ReadUint32(),
				Losses:  r.ReadUint32(),
				Points:  r.ReadUint32(),
			}
		}

		m.players = make([]MiniGameRoomPlayer, 0, len(avatars))
		for _, a := range avatars {
			m.players = append(m.players, MiniGameRoomPlayer{
				Slot:   a.slot,
				Avatar: a.avatar,
				Name:   a.name,
				Record: records[a.slot],
			})
		}

		m.title = r.ReadAsciiString()
		m.gameKind = r.ReadByte()
		m.tournament = r.ReadBool()
		if m.tournament {
			m.round = r.ReadByte()
		}
	}
}
