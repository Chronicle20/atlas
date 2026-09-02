package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CharacterMoveHandle = "CharacterMoveHandle"

// moveDrBlocks reports whether MOVE_PLAYER carries the self-move anti-cheat
// header words (dr0/dr1 before fieldKey, dr2/dr3 after it, dwKey/crc32 after
// the move crc). GMS v84+ — IDA-verified against the v84 senders
// CVecCtrlUser::EndUpdateActive (sub_A1334E) and the keyboard/teleport sender
// (sub_9843EA); v83 (@0x9cb992) writes only fieldKey+crc.
//
// jms v185 carries them as well. This is live-capture derived, not
// decompile-derived: the jms sender for this op is the registry-primary
// CUserLocal::OnKey, which is not decompilable in the retail SCY dump (the
// same code-flow virtualization that hides the attack senders' encode tails).
// Nine consecutive MOVE_PLAYER frames captured from the JMS 185.1 tenant
// (2026-09-02, lengths 90/98/108/108/116/126/134/144/188) all place the
// CMovePath blob at frame offset 0x1f — i.e. behind a 29-byte header of
// exactly this shape — and under that alignment every frame yields a sensible
// origin x/y plus an element count that tracks the frame length (2,3,3,4,4,5,
// 5,8). See docs/tasks/fix-jms185-attack-decode/diagnosis.md.
//
// A prior pass gated these to GMS only, citing
// CVecCtrlUser::EndUpdateActive @0xaaa076 ("Encode1(detectFlag) then, if
// active, Encode1(fieldKey)+Encode4(crc)+CMovePath::Flush — no dr fields").
// That function decompiles, but it is NOT the sender that produced any
// observed frame: its tail is a 6-byte bActive/fieldKey/crc run, whereas the
// six bytes preceding the blob on the wire are 00 00 followed by a 4-byte
// word, and its bActive branch cannot explain two equal-length frames whose
// byte at that offset differs (0x01 vs 0x00). It is a second, unused send
// path; the conclusion drawn from it was applied to the wrong function.
func moveDrBlocks(t tenant.Model) bool {
	return (t.IsRegion("GMS") && t.MajorAtLeast(84)) || t.IsRegion("JMS")
}

// moveCrc reports whether MOVE_PLAYER carries the field move-CRC word between
// the dr2/dr3 pair and dwKey. GMS v72+ (v61's sub_801109 @0x8012a7 writes
// fieldKey+Flush with no crc; v72 @0x8cb63e writes it) and jms v185, which
// carries the full header — see moveDrBlocks.
func moveCrc(t tenant.Model) bool {
	return (t.IsRegion("GMS") && t.MajorAtLeast(72)) || t.IsRegion("JMS")
}

type Move struct {
	dr0      uint32
	dr1      uint32
	fieldKey byte
	dr2      uint32
	dr3      uint32
	crc      uint32
	dwKey    uint32
	crc32    uint32
	movement model.Movement
}

func (m Move) Dr0() uint32                  { return m.dr0 }
func (m Move) Dr1() uint32                  { return m.dr1 }
func (m Move) FieldKey() byte               { return m.fieldKey }
func (m Move) Dr2() uint32                  { return m.dr2 }
func (m Move) Dr3() uint32                  { return m.dr3 }
func (m Move) Crc() uint32                  { return m.crc }
func (m Move) DwKey() uint32                { return m.dwKey }
func (m Move) Crc32() uint32                { return m.crc32 }
func (m Move) MovementData() model.Movement { return m.movement }

func (m Move) Operation() string {
	return CharacterMoveHandle
}

func (m Move) String() string {
	return fmt.Sprintf("dr0 [%d] dr1 [%d] fieldKey [%d] dr2 [%d] dr3 [%d] crc [%d] dwKey [%d] crc32 [%d] elements [%d]",
		m.dr0, m.dr1, m.fieldKey, m.dr2, m.dr3, m.crc, m.dwKey, m.crc32, len(m.movement.Elements))
}

// Encode writes the movement packet.
//
// Version gating lives in moveDrBlocks / moveCrc; see those for the per-version
// evidence, including why the earlier "JMS uses the GMS v83-style layout (no dr
// fields)" reading of CVecCtrlUser::EndUpdateActive@0xaaa076 was drawn from a
// send path the client does not use for this op.
func (m Move) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		// dr0/dr1/dr2/dr3/dwKey/crc32 are the GMS self-move anti-cheat header.
		// CONFIRMED v84+ against the v84 client: both self-move senders
		// CVecCtrlUser::EndUpdateActive (sub_A1334E) and the keyboard/teleport
		// sender (sub_9843EA) write Encode4(dr0) Encode4(dr1) Encode1(fieldKey)
		// Encode4(dr2) Encode4(dr3) Encode4(crc) Encode4(dwKey) Encode4(crc32)
		// before CMovePath::Flush. v83 (CVecCtrlUser::EndUpdateActive @0x9cb992)
		// writes only fieldKey+crc. So the boundary is v84, not v87 — the prior
		// >=87 gate skipped 24 header bytes on v84 and desynced every move packet.
		if moveDrBlocks(t) {
			w.WriteInt(m.dr0)
			w.WriteInt(m.dr1)
		}
		w.WriteByte(m.fieldKey)
		if moveDrBlocks(t) {
			w.WriteInt(m.dr2)
			w.WriteInt(m.dr3)
		}
		// The move CRC (get_field()+476) is CONFIRMED absent on the very-legacy
		// GMS v61 sender: CUserLocal move-flush sub_801109 (@0x8012a7) builds
		// COutPacket(38) = Encode1(fieldKey) + CMovePath::Flush with NO Encode4(crc)
		// between them, whereas v72 CVecCtrlUser::EndUpdateActive @0x8cb63e writes
		// Encode1(fieldKey)+Encode4(crc)+Flush. The prior >28 gate assumed crc from
		// v29; the verified boundary is v72 (v61 has none, v72 does). Gate to >=72 so
		// v61 emits fieldKey+movement only; v72+/jms layouts are unchanged.
		if moveCrc(t) {
			w.WriteInt(m.crc)
		}
		if moveDrBlocks(t) {
			w.WriteInt(m.dwKey)
			w.WriteInt(m.crc32)
		}
		w.WriteByteArray(m.movement.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *Move) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		// Mirror of Encode: dr0/dr1/dr2/dr3/dwKey/crc32 are CONFIRMED v84+ against the
		// v84 client self-move senders (sub_A1334E, sub_9843EA); v83 sends only
		// fieldKey+crc; jms v185 carries the full header (moveDrBlocks). Must stay
		// textually identical to Encode.
		if moveDrBlocks(t) {
			m.dr0 = r.ReadUint32()
			m.dr1 = r.ReadUint32()
		}
		m.fieldKey = r.ReadByte()
		if moveDrBlocks(t) {
			m.dr2 = r.ReadUint32()
			m.dr3 = r.ReadUint32()
		}
		// Mirror of Encode: the move CRC is absent on GMS v61 (sub_801109 @0x8012a7
		// writes no Encode4(crc)); verified boundary is v72. Gate to >=72.
		if moveCrc(t) {
			m.crc = r.ReadUint32()
		}
		if moveDrBlocks(t) {
			m.dwKey = r.ReadUint32()
			m.crc32 = r.ReadUint32()
		}
		m.movement.Decode(l, ctx)(r, options)
	}
}
