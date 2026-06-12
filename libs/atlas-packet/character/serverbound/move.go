package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterMoveHandle = "CharacterMoveHandle"

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

func (m Move) Dr0() uint32                    { return m.dr0 }
func (m Move) Dr1() uint32                    { return m.dr1 }
func (m Move) FieldKey() byte                 { return m.fieldKey }
func (m Move) Dr2() uint32                    { return m.dr2 }
func (m Move) Dr3() uint32                    { return m.dr3 }
func (m Move) Crc() uint32                    { return m.crc }
func (m Move) DwKey() uint32                  { return m.dwKey }
func (m Move) Crc32() uint32                  { return m.crc32 }
func (m Move) MovementData() model.Movement   { return m.movement }

func (m Move) Operation() string {
	return CharacterMoveHandle
}

func (m Move) String() string {
	return fmt.Sprintf("dr0 [%d] dr1 [%d] fieldKey [%d] dr2 [%d] dr3 [%d] crc [%d] dwKey [%d] crc32 [%d] elements [%d]",
		m.dr0, m.dr1, m.fieldKey, m.dr2, m.dr3, m.crc, m.dwKey, m.crc32, len(m.movement.Elements))
}

// Encode writes the movement packet.
//
// IDA JMS v185 CVecCtrlUser::EndUpdateActive@0xaaa076: encodes Encode1(detectFlag) then if active:
// Encode1(fieldKey)+Encode4(crc)+CMovePath::Flush — NO dr0/dr1/dr2/dr3/dwKey/crc32 fields.
// The || JMS clause on dr-field gates was incorrect; JMS uses GMS v83-style layout (no dr fields).
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
		if t.IsRegion("GMS") && t.MajorAtLeast(84) {
			w.WriteInt(m.dr0)
			w.WriteInt(m.dr1)
		}
		w.WriteByte(m.fieldKey)
		if t.IsRegion("GMS") && t.MajorAtLeast(84) {
			w.WriteInt(m.dr2)
			w.WriteInt(m.dr3)
		}
		if t.Region() == "GMS" && t.MajorVersion() > 28 {
			w.WriteInt(m.crc)
		}
		if t.IsRegion("GMS") && t.MajorAtLeast(84) {
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
		// v84 client self-move senders (sub_A1334E, sub_9843EA). v83 sends only
		// fieldKey+crc. JMS (CVecCtrlUser::EndUpdateActive@0xaaa076) has no dr fields,
		// so it stays on the v83 layout. Must stay textually identical to Encode.
		if t.IsRegion("GMS") && t.MajorAtLeast(84) {
			m.dr0 = r.ReadUint32()
			m.dr1 = r.ReadUint32()
		}
		m.fieldKey = r.ReadByte()
		if t.IsRegion("GMS") && t.MajorAtLeast(84) {
			m.dr2 = r.ReadUint32()
			m.dr3 = r.ReadUint32()
		}
		if t.Region() == "GMS" && t.MajorVersion() > 28 {
			m.crc = r.ReadUint32()
		}
		if t.IsRegion("GMS") && t.MajorAtLeast(84) {
			m.dwKey = r.ReadUint32()
			m.crc32 = r.ReadUint32()
		}
		m.movement.Decode(l, ctx)(r, options)
	}
}
