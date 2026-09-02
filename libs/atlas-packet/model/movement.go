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

// XOffset/YOffset on an absolute-position (NORMAL) movement fragment are
// DIRECTIONAL: the boundary for reading them off the wire is not the boundary
// for writing them back. GMS v87 reads them and does not write them, so the two
// gates below are deliberately different and must NOT be collapsed again.
//
// In the client the pair sits at CMovePath::ELEM +0x14/+0x16 (named xOffset /
// yOffset outright in the v95 symbols, ELEM decoded at CMovePath::Decode
// @0x667920). It is written/read only for the absolute-position attrs — the
// case 0/5/15/17 arm — immediately after fh and the attr-15 fhFallStart, and
// before the common bMoveAction/tElapse tail.
//
// Per-version, from the client's own CMovePath pair:
//
//	version   Encode (client -> server)   Decode (server -> client)
//	GMS v83   @0x68a563  no               @0x68a33c  no
//	GMS v84   @0x6a0fd0  no               @0x6a0fd0  no
//	GMS v87   @0x6c70fe  YES              @0x6c6e86  NO      <- asymmetric
//	GMS v92   @0x65a260  YES              @0x65ad60  yes
//	GMS v95   @0x666e20  YES              @0x667920  yes
//	JMS v185  @0x70b6c4  YES              @0x70b3ce  yes
//
// v87's Decode is 154 instructions ending at 0x6c709a (0x6c709a onward is the
// jump table, so this is the whole function) and its absolute arm reads
// x/y/vx/vy/fh, the attr-15 fhFallStart, then goes straight to the
// bMoveAction/tElapse tail. Its Encode at 6c720a/6c7218 does `mov ax,[edi+14h]`
// and `mov ax,[edi+16h]` and writes both. Every other version listed reads back
// what it wrote.
//
// Consequence, and the reason this file has been wrong in both directions: on
// v87 a server that echoes the pair back makes the client read xOffset's low
// byte as bMoveAction and yOffset as tElapse, then read the real
// bMoveAction/tElapse as the NEXT fragment's attr and body. NPCs, mobs and
// remote players then teleport instead of walking.
//
// Do not confuse this pair with the one guarded by CClientOptMan::GetOpt(..., 2)
// four bytes later at +0x18/+0x1A. The v95 symbols name those usRandCnt /
// usActualRandCnt — move-rand counters, present on EVERY fragment type, not just
// the absolute ones, and symmetric in Encode and Decode. Atlas neither reads nor
// writes them, which is consistent with that runtime option being off for the
// clients Atlas serves; an earlier revision of this comment mistook them for the
// offsets and recorded a "runtime option" caveat that does not apply here.
//
// movementElementOffsetsInbound gates the read. GMS v87+ and every non-GMS
// region; v83/v84 have no such field on either side.
//
// The v87 read is confirmed empirically as well as by disassembly: eight
// distinct live v87 monster-move frames captured by logUnconfiguredMovementCode
// were replayed against both element-size models. With the pair present all 8
// parse cleanly end-to-end as all-NORMAL fragments; without it, 1 of 8 parses
// and that one only coincidentally. The failure signature in the field was every
// EVEN fragment failing at an exact 18-byte stride: 18 == 1 attr + 10 coords +
// 4 offsets + 3 tail (task-218 field reports #3/#5).
func movementElementOffsetsInbound(t tenant.Model) bool {
	return !t.IsRegion("GMS") || t.MajorAtLeast(87)
}

// movementElementOffsetsOutbound gates the write, and excludes GMS v87 for the
// reason given above.
//
// Boundary 92 rather than 88: v87 is IDA-verified not to read the pair and v92
// is IDA-verified to read it; v88..v91 have no IDB, and deploy/k8s/base/
// versions.json ships no GMS version between 87 and 92, so every value in
// 88..92 is behaviourally identical for any tenant Atlas can serve. 92 is the
// lowest one backed by direct evidence.
func movementElementOffsetsOutbound(t tenant.Model) bool {
	return !t.IsRegion("GMS") || t.MajorAtLeast(92)
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
		// XOffset/YOffset on NORMAL elements. This gate is deliberately NOT the
		// one Encode uses — see movementElementOffsetsInbound/Outbound. GMS v87
		// sends the pair and does not read it back.
		if movementElementOffsetsInbound(t) {
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
		// Deliberately a different gate from Decode's: GMS v87's
		// CMovePath::Decode @0x6c6e86 never reads this pair, so writing it back
		// desyncs the client's fragment loop.
		if movementElementOffsetsOutbound(t) {
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
