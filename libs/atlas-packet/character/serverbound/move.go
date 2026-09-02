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

// moveKeyPadTail reports whether serverbound MOVE_PLAYER ends with the keypad
// history + path bounding rectangle that CMovePath::Encode appends after the
// element loop.
//
// IDA-verified for jms v185, CMovePath::Encode @0x70b6c4 (reached from
// CMovePath::Flush @0x70ba2c, which delegates all encoding to it):
//
//	@0x70b8ec  Encode1(len(m_aKeyPadState))        -- entry COUNT, not byte count
//	@0x70b8f3  loop i += 2, Encode1(nType) where
//	           nType = state[i]&0xF, and for every i except the last
//	           nType |= state[i+1]<<4                -- two entries per byte
//	@0x70b942  Encode2(m_rcMove.left)
//	@0x70b950  Encode2(m_rcMove.top)
//	@0x70b95e  Encode2(m_rcMove.right)
//	@0x70b96c  Encode2(m_rcMove.bottom)
//
// so the tail is 1 + ceil(count/2) + 8 bytes. Both jms captures carry
// count = 0x11 = 17, giving 1 + 9 + 8 = 18 -- exactly the trailing bytes the
// decoder was leaving unread, and the rect decodes to the real bounds of the
// walk (capture two_elements: left -77, top 215, right -71, bottom 215).
//
// Nine independent read-only IDA investigations (one per remaining GMS IDB;
// see docs/tasks/fix-jms185-attack-decode/movepath-tail-findings.md) confirm
// CMovePath::Encode appends the identical tail -- same field order, same
// 1 + ceil(count/2) + 8 size formula -- on every other supported client:
//
//	gms_v48  Encode @0x56201a  count @0x5621bd  nibbles 0x5621c2-0x5621f9  rect 0x56220b/0x562219/0x562227/0x562235
//	gms_v61  Encode @0x5e298d  count @0x5e2b72  nibbles 0x5e2b8d-0x5e2bae  rect 0x5e2bc0/0x5e2bce/0x5e2bdc/0x5e2bef
//	gms_v72  Encode @0x634ddb  count @0x634fc0  nibbles 0x634fdb-0x634ffc  rect 0x63500e/0x63501c/0x63502a/0x635038
//	gms_v79  Encode @0x6575fa  count @0x6577df  nibbles 0x6577e4-0x65781b  rect 0x65782d/0x65783b/0x657849/0x65785c
//	gms_v83  Encode @0x68a563  count @0x68a748  nibbles 0x68a763-0x68a784  rect 0x68a796/0x68a7a4/0x68a7b2/0x68a7c0
//	gms_v84  Encode @0x6a121a  count @0x6a141e  nibbles 0x6a1423-0x6a145a  rect 0x6a146c/0x6a147a/0x6a1488/0x6a1496
//	gms_v87  Encode @0x6c70fe  count @0x6c734c  nibbles 0x6c7383-0x6c738b  rect 0x6c73a2/0x6c73b0/0x6c73be/0x6c73cc
//	gms_v92  Encode @0x65a260  count @0x65a61b  nibbles 0x65a620-0x65a68c  rect 0x65a6da/0x65a71c/0x65a75e/0x65a7a1
//	gms_v95  Encode @0x666e20  count @0x6671db  nibbles 0x6671e0-0x66724c  rect 0x66729a/0x6672dc/0x66731e/0x667361
//	jms_v185 Encode @0x70b6c4  count @0x70b8ec  nibbles 0x70b8f3-0x70b92b  rect 0x70b942/0x70b950/0x70b95e/0x70b96c
//
// v48 and v84 have no exported symbol for the encoder; both were located
// structurally as the function CMovePath::Flush delegates to, anchored on the
// Encode2/Encode2/Encode1-header-then-per-element-switch shape. On v92/v95
// Hex-Rays inlines Encode1/Encode2 into direct buffer stores; the addresses
// above are the store sites, not call sites. So THE TAIL IS UNCONDITIONAL ON
// SERVERBOUND MOVE_PLAYER ACROSS ALL TEN SUPPORTED CLIENTS.
//
// THE TAIL IS SERVERBOUND-ONLY, which is why it must live here and not in the
// shared model.Movement. The client's read side, CMovePath::Decode, consumes
// the keypad block and the rect ONLY under `if (bPassive)` -- but that gate is
// implemented three different ways across the ten clients, not one:
//
//   - gms_v92, gms_v95, jms_v185: a live bPassive parameter. The keypad/rect
//     block sits inside `if (bPassive)`, and the clientbound entry point
//     (CUserRemote::OnMove, jms @0xa443ee) calls
//     CMovePath::OnMovePacket(..., iPacket, 0) -- bPassive = 0 -- so the
//     clientbound decode path never reads the tail.
//   - gms_v48, gms_v72, gms_v83, gms_v84, gms_v87: no such parameter is
//     compiled into the interface at all -- `retn 4`, one stack arg, every
//     call site pushes exactly one value -- and Decode contains no
//     tail-reading instructions whatsoever. There is nothing to gate.
//   - gms_v79: the parameter exists in the mangled signature but is dead --
//     it is clobbered as scratch at @0x657400 and reused as a "previous X"
//     cache, so the optimizer never materializes a real bPassive value at any
//     call site. The tail is unreachable code, not a live-gated read.
//
// All three mechanisms reach the same conclusion (serverbound-only, tail
// never read on the clientbound broadcast path), but they are NOT the same
// mechanism -- do not describe v48/v72/v79/v83/v84/v87 as "same as jms".
// Adding these fields to model.Movement would make every clientbound movement
// broadcast emit bytes the client never reads on any version.
func moveKeyPadTail(_ tenant.Model) bool {
	return true
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
	// keyPadStates is the client's per-move keypad history, one 4-bit value
	// per entry (see moveKeyPadTail). Nil on versions without the tail.
	keyPadStates []byte
	// moveRect is the bounding rectangle of the whole path: the origin
	// extended by every element's position (CMovePath::Encode maintains
	// m_rcMove as it walks the elements, @0x70b81c-0x70b842).
	moveRectLeft   int16
	moveRectTop    int16
	moveRectRight  int16
	moveRectBottom int16
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
func (m Move) KeyPadStates() []byte         { return m.keyPadStates }

// MoveRect reports the path bounding rectangle the client appends after the
// keypad block: left, top, right, bottom.
func (m Move) MoveRect() (int16, int16, int16, int16) {
	return m.moveRectLeft, m.moveRectTop, m.moveRectRight, m.moveRectBottom
}

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
		// Keypad history + path bounding rect — see moveKeyPadTail. Two 4-bit
		// entries per byte, low nibble first; the final byte of an odd-length
		// run carries only the low nibble.
		if moveKeyPadTail(t) {
			w.WriteByte(byte(len(m.keyPadStates)))
			for i := 0; i < len(m.keyPadStates); i += 2 {
				b := m.keyPadStates[i] & 0x0F
				if i != len(m.keyPadStates)-1 {
					b |= m.keyPadStates[i+1] << 4
				}
				w.WriteByte(b)
			}
			w.WriteShort(uint16(m.moveRectLeft))
			w.WriteShort(uint16(m.moveRectTop))
			w.WriteShort(uint16(m.moveRectRight))
			w.WriteShort(uint16(m.moveRectBottom))
		}
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
		// Mirror of Encode: keypad history + path bounding rect, serverbound
		// only (CMovePath::Decode reads them under bPassive, which the
		// clientbound OnMove path passes as 0). See moveKeyPadTail.
		if moveKeyPadTail(t) {
			count := int(r.ReadByte())
			states := make([]byte, 0, count)
			var packed byte
			for i := 0; i < count; i++ {
				if i%2 == 0 {
					packed = r.ReadByte()
				} else {
					packed >>= 4
				}
				states = append(states, packed&0x0F)
			}
			m.keyPadStates = states
			m.moveRectLeft = r.ReadInt16()
			m.moveRectTop = r.ReadInt16()
			m.moveRectRight = r.ReadInt16()
			m.moveRectBottom = r.ReadInt16()
		}
	}
}
