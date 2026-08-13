package model

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	TypeNormal        = "NORMAL"
	TypeTeleport      = "TELEPORT"
	TypeStartFallDown = "START_FALL_DOWN"
	TypeFlyingBlock   = "FLYING_BLOCK"
	TypeJump          = "JUMP"
	TypeStatChange    = "STAT_CHANGE"
)

type MovementCodec interface {
	packet.Codec
	EncodeType(w *response.Writer)
}

// gmsMovementElementOffsets reports whether a NORMAL movement fragment carries
// the trailing XOffset/YOffset pair. GMS v87+, and every non-GMS region.
//
// This was previously v88+, sharing a boundary with Movement's StartVx/StartVy
// on the assumption that one client rework introduced both. It did not, and the
// conflation cost the whole "Code [253/254/255] not configured for use in
// movement" flood on v87 — thousands per minute, with the server's own log
// warning it would crash the client (task-218 field reports #3/#5).
//
// The two fields have DIFFERENT boundaries:
//
//   - StartVx/StartVy: v87 CMovePath::Encode @0x6c70fe writes Encode2(x),
//     Encode2(y), Encode1(count) and nothing else, so v87 does NOT have them.
//     That gate correctly stays at 88 (see Movement.Decode).
//   - XOffset/YOffset: v87 writes them per element. v83 @0x68a563 and v84
//     @0x6a0fd0 have no such read/write at all; v87 @0x6c70fe (Encode) and
//     @0x6c6e86 (Decode) both carry the pair.
//
// Confirmed empirically as well as by disassembly, which is what settled it:
// eight distinct live v87 monster-move frames captured by
// logUnconfiguredMovementCode were replayed against both element-size models.
// With the pair present all 8 parse cleanly end-to-end as all-NORMAL elements;
// without it, 1 of 8 parses and that one only coincidentally. The failure
// signature in the field was every EVEN element failing at an exact 18-byte
// stride: 18 == 1 type + 10 coords + 4 offsets + 3 tail, i.e. the decoder read
// 14 for a real 18-byte element and then consumed the 4-byte remainder as a
// phantom element, re-syncing on every second one.
//
// CAVEAT, deliberately recorded rather than hidden: in the client this pair is
// gated on CClientOptMan::GetOpt(..., 2) — a RUNTIME option, not a version. A
// server cannot observe it, so a version gate is the best available
// approximation and matches what every client Atlas serves actually does. The
// same option also gates three extra Decode4 (a move-rand seed) in
// CMobPool::OnMobChangeController @0x6b52c3, which Atlas does NOT send; that
// packet nonetheless works in the field, so the option's exact scope is not
// fully understood. Do not "fix" the control packet to match without evidence.
func gmsMovementElementOffsets(t tenant.Model) bool {
	return !t.IsRegion("GMS") || t.MajorAtLeast(87)
}

type Movement struct {
	StartX int16
	StartY int16
	// StartVx/StartVy are GMS v88+ only — see the gate in Decode/Encode.
	StartVx  int16
	StartVy  int16
	Elements []MovementCodec
}

func (m *Movement) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.StartX = r.ReadInt16()
		m.StartY = r.ReadInt16()

		// StartVx/StartVy are GMS v88+ — the same client movement rework that
		// added XOffset/YOffset to NormalElement (see the gate at NormalElement
		// .Decode). v83/v84/v87 and JMS write x,y,count only:
		//   v83 CMovePath::Encode@0x68a563, v87 @0x6c70fe, jms @0x70b6c4 — 2 Encode2 + Encode1.
		//   v92 @0x65a260, v95 @0x666e20 — 4 Encode2 + Encode1.
		// NOTE the predicate shape: this is IsRegion("GMS") && MajorAtLeast(88),
		// NOT the !IsRegion("GMS") || MajorAtLeast(88) shape used for
		// XOffset/YOffset. JMS v185 was checked directly (@0x70b6c4) and writes
		// the TWO-field header, so JMS is EXCLUDED here even though it is
		// INCLUDED by the XOffset gate. Reusing that predicate by reflex breaks
		// JMS movement.
		//
		// Boundary 88 vs 92: observed v87 no, v92 yes; v88..v91 have no IDB.
		// 88 is chosen for consistency with the adjacent XOffset/YOffset gate,
		// which pins the same rework — and deploy/k8s/base/versions.json ships
		// no GMS version between 87 and 92, so the two are behaviourally
		// indistinguishable for every tenant Atlas can serve.
		//
		// MUST stay textually identical to Encode.
		if t.IsRegion("GMS") && t.MajorAtLeast(88) {
			m.StartVx = r.ReadInt16()
			m.StartVy = r.ReadInt16()
		}

		numElems := r.ReadByte()
		elems := make([]MovementCodec, numElems)
		for i := byte(0); i < numElems; i++ {
			var elem MovementCodec
			elemType := r.ReadByte()

			// One resolution per element, purely to decide whether to report a
			// misalignment. The dispatch below re-asks per candidate kind, which
			// is cheap and keeps the existing structure; what must NOT happen is
			// the lookup logging on every one of those asks.
			if _, _, ok := resolveMovementPathAttr(elemType, options); !ok {
				logUnconfiguredMovementCode(l, r, elemType, i, numElems)
			}

			if isMovementType(l)(elemType, options, TypeNormal) {
				elem = &NormalElement{Element{ElemType: elemType, StartX: m.StartX, StartY: m.StartY}}
			} else if isMovementType(l)(elemType, options, TypeTeleport) {
				elem = &TeleportElement{Element{ElemType: elemType, StartX: m.StartX, StartY: m.StartY}}
			} else if isMovementType(l)(elemType, options, TypeStartFallDown) {
				elem = &StartFallDownElement{Element{ElemType: elemType, StartX: m.StartX, StartY: m.StartY}}
			} else if isMovementType(l)(elemType, options, TypeFlyingBlock) {
				elem = &FlyingBlockElement{Element{ElemType: elemType, StartX: m.StartX, StartY: m.StartY}}
			} else if isMovementType(l)(elemType, options, TypeJump) {
				elem = &JumpElement{Element{ElemType: elemType, StartX: m.StartX, StartY: m.StartY}}
			} else if isMovementType(l)(elemType, options, TypeStatChange) {
				elem = &StatChangeElement{Element{ElemType: elemType, StartX: m.StartX, StartY: m.StartY}}
			} else {
				elem = &Element{ElemType: elemType}
			}
			elem.Decode(l, ctx)(r, options)
			elems[i] = elem
		}
		m.Elements = elems
	}
}

type Element struct {
	StartX      int16
	StartY      int16
	BMoveAction byte
	BStat       byte
	X           int16
	Y           int16
	Vx          int16
	Vy          int16
	Fh          int16
	FhFallStart int16
	XOffset     int16
	YOffset     int16
	TElapse     int16
	ElemType    byte
}

func (m *Element) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.BMoveAction = r.ReadByte()
		m.TElapse = r.ReadInt16()
	}
}

func (m *Element) EncodeType(w *response.Writer) {
	w.WriteByte(m.ElemType)
}

type NormalElement struct {
	Element
}

type TeleportElement struct {
	Element
}

type StartFallDownElement struct {
	Element
}

type FlyingBlockElement struct {
	Element
}

type JumpElement struct {
	Element
}

type StatChangeElement struct {
	Element
}

func (m *NormalElement) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.X = r.ReadInt16()
		m.Y = r.ReadInt16()
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.Fh = r.ReadInt16()
		if isMovementName(l)(m.ElemType, options, "FALL_DOWN") {
			m.FhFallStart = r.ReadInt16()
		}
		// XOffset/YOffset on NORMAL elements — see gmsMovementElementOffsets.
		// MUST stay textually identical to Encode or Atlas corrupts its own
		// movement packets.
		if gmsMovementElementOffsets(t) {
			m.XOffset = r.ReadInt16()
			m.YOffset = r.ReadInt16()
		}
		m.Element.Decode(l, ctx)(r, options)
	}
}

func (m *TeleportElement) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.X = r.ReadInt16()
		m.Y = r.ReadInt16()
		m.Fh = r.ReadInt16()
		m.Element.Decode(l, ctx)(r, options)
	}
}

func (m *StartFallDownElement) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.X = m.StartX
		m.Y = m.StartY
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.FhFallStart = r.ReadInt16()
		m.Element.Decode(l, ctx)(r, options)
	}
}

func (m *FlyingBlockElement) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.X = r.ReadInt16()
		m.Y = r.ReadInt16()
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.Element.Decode(l, ctx)(r, options)
	}
}

func (m *JumpElement) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.X = m.StartX
		m.Y = m.StartY
		m.Vx = r.ReadInt16()
		m.Vy = r.ReadInt16()
		m.Element.Decode(l, ctx)(r, options)
	}
}

func (m *StatChangeElement) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.BStat = r.ReadByte()
	}
}

func (m *Movement) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.StartX)
		w.WriteInt16(m.StartY)
		// StartVx/StartVy are GMS v88+. Paired with the Decode boundary; the two
		// MUST stay textually identical. JMS is EXCLUDED (jms CMovePath::Encode
		// @0x70b6c4 writes the two-field header) — do not reuse the
		// XOffset/YOffset predicate shape here.
		if t.IsRegion("GMS") && t.MajorAtLeast(88) {
			w.WriteInt16(m.StartVx)
			w.WriteInt16(m.StartVy)
		}
		w.WriteByte(byte(len(m.Elements)))
		for _, element := range m.Elements {
			element.EncodeType(w)
			w.WriteByteArray(element.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}

func (m *Element) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(_ map[string]interface{}) []byte {
		w.WriteByte(m.BMoveAction)
		w.WriteInt16(m.TElapse)
		return w.Bytes()
	}
}

func (m *NormalElement) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.X)
		w.WriteInt16(m.Y)
		w.WriteInt16(m.Vx)
		w.WriteInt16(m.Vy)
		w.WriteInt16(m.Fh)
		if isMovementName(l)(m.ElemType, options, "FALL_DOWN") {
			w.WriteInt16(m.FhFallStart)
		}
		// Paired with the Decode boundary; the two MUST stay textually identical.
		if gmsMovementElementOffsets(t) {
			w.WriteInt16(m.XOffset)
			w.WriteInt16(m.YOffset)
		}
		w.WriteByteArray(m.Element.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *TeleportElement) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.X)
		w.WriteInt16(m.Y)
		w.WriteInt16(m.Fh)
		w.WriteByteArray(m.Element.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *StartFallDownElement) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.Vx)
		w.WriteInt16(m.Vy)
		w.WriteInt16(m.FhFallStart)
		w.WriteByteArray(m.Element.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *FlyingBlockElement) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.X)
		w.WriteInt16(m.Y)
		w.WriteInt16(m.Vx)
		w.WriteInt16(m.Vy)
		w.WriteByteArray(m.Element.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *JumpElement) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.Vx)
		w.WriteInt16(m.Vy)
		w.WriteByteArray(m.Element.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *StatChangeElement) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(_ map[string]interface{}) []byte {
		w.WriteByte(m.BStat)
		return w.Bytes()
	}
}

// movementPathAttrFromOptions resolves a movement fragment's type code against
// the tenant's configured `types` table, reporting whether the lookup succeeded.
//
// It deliberately does NOT log. Decode asks this question up to six times per
// element (once per candidate element kind) plus once more for the FALL_DOWN
// name check, so logging here turned ONE unconfigured code into six or seven
// identical lines and buried the only fact worth having — the bytes. Decode now
// resolves once per element and calls logUnconfiguredMovementCode on failure.
func movementPathAttrFromOptions(_ logrus.FieldLogger) func(attr byte, options map[string]interface{}) (string, string) {
	return func(attr byte, options map[string]interface{}) (string, string) {
		name, kind, _ := resolveMovementPathAttr(attr, options)
		return name, kind
	}
}

func resolveMovementPathAttr(attr byte, options map[string]interface{}) (string, string, bool) {
	genericCodes, ok := options["types"]
	if !ok {
		return "NOT_FOUND", "DEFAULT", false
	}

	codes, ok := genericCodes.([]interface{})
	if !ok {
		return "NOT_FOUND", "DEFAULT", false
	}

	if len(codes) == 0 || int(attr) >= len(codes) {
		return "NOT_FOUND", "DEFAULT", false
	}

	theType, ok := codes[attr].(map[string]interface{})
	if !ok {
		return "NOT_FOUND", "DEFAULT", false
	}

	return theType["Name"].(string), theType["Type"].(string), true
}

// movementDiagnosticDumpLimit caps the hex dump below. A movement packet is a
// few hundred bytes at most; the cap only guards against an absurd frame.
const movementDiagnosticDumpLimit = 512

// logUnconfiguredMovementCode reports a movement fragment whose type code is not
// in the tenant's table, WITH the bytes needed to diagnose it.
//
// A code outside the table is almost never a genuinely unknown fragment type —
// the client only emits codes it has arms for. It means the reader is no longer
// on a fragment boundary, i.e. some earlier field was decoded at the wrong
// width for this client version. Diagnosing that needs the frame, the offset
// the reader had reached, and which element of how many failed; without them
// the message says only that something drifted, which is what made the GMS v87
// occurrence (task-218 field reports #3/#5) unactionable for so long.
func logUnconfiguredMovementCode(l logrus.FieldLogger, r *request.Reader, attr byte, index byte, count byte) {
	buf := r.GetBuffer()
	truncated := ""
	if len(buf) > movementDiagnosticDumpLimit {
		buf = buf[:movementDiagnosticDumpLimit]
		truncated = " (truncated)"
	}
	l.WithFields(logrus.Fields{
		"movement.code":          attr,
		"movement.elementIndex":  index,
		"movement.elementCount":  count,
		"movement.readerOffset":  r.Position(),
		"movement.bytesRemained": r.Available(),
	}).Errorf("Code [%d] not configured for use in movement (element %d of %d, reader at offset %d). "+
		"This is a decode misalignment, not an unknown fragment type: the client only sends codes it has arms for. "+
		"Frame%s: %x", attr, index+1, count, r.Position(), truncated, buf)
}

func isMovementType(l logrus.FieldLogger) func(reference byte, options map[string]interface{}, movementType string) bool {
	return func(reference byte, options map[string]interface{}, movementType string) bool {
		_, t := movementPathAttrFromOptions(l)(reference, options)
		return t == movementType
	}
}

func isMovementName(l logrus.FieldLogger) func(reference byte, options map[string]interface{}, movementName string) bool {
	return func(reference byte, options map[string]interface{}, movementName string) bool {
		n, _ := movementPathAttrFromOptions(l)(reference, options)
		return n == movementName
	}
}
