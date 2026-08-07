package clientbound

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const AffectedAreaCreatedWriter = "AffectedAreaCreated"

// trailingWidth selects how the nElemAttr word of the body is written. It is
// always the FIRST of the trailing words and always feeds the affected-area
// record's +0x30 slot (AFFECTEDAREA.nElemAttr); the optional second trailing
// word (nPhase, +0x48) exists only on GMS v92+. See the version gates in
// AffectedAreaCreated.Encode.
type trailingWidth int

const (
	trailingWidthWide trailingWidth = iota
	trailingWidthByte
	trailingWidthNone
)

// AffectedAreaCreated is the clientbound packet that announces a new
// affected-area (mist) on the field. The wire body follows the client read-order
// in CAffectedAreaPool::OnAffectedAreaCreated:
//
//	Decode4 dwId, Decode4 nType, Decode4 dwOwnerId, Decode4 nSkillID,
//	Decode1 nSLV, Decode2 skillDelay, DecodeBuf(16) rcArea (4×int32 absolute
//	RECT), Decode4 nElemAttr, [Decode4 nPhase — GMS v92+].
//
// Field semantics, read from the client's own record layout. AFFECTEDAREA is
// 76 bytes (gms_v95 PDB-backed): dwID 0x0, nType 0x4, dwOwnerID 0x8,
// nSkillID 0xC, nSLV 0x10, tStart 0x14, tEnd 0x18, bEnd 0x1C, rcArea 0x20,
// nElemAttr 0x30, nDamage 0x34, bLeftToDraw 0x38, nFogState 0x3C,
// apLayer 0x40, posList 0x44, nPhase 0x48.
//
//   - skillDelay (Decode2) is NOT a lifetime. The client computes
//     tStart = get_update_time() + 100*skillDelay (v83 @0x431b50,
//     v95 @0x437fa3), and CAffectedAreaPool::Update gates the mist's FIRST
//     DRAW on it: `if (tStart && tCur - tStart >= 0) { FindAndDraw();
//     tStart = 0; }` (v83 @0x431214-0x431238, v95 @0x4369b8). It is therefore
//     a delay-before-drawing in units of 100 ms; a non-zero value hides the
//     mist for that long. Send 0 to draw immediately.
//   - The first trailing Decode4 lands in nElemAttr (+0x30) — v83 @0x431b3b,
//     v95 @0x437fd9. It is not a time field.
//   - The second trailing Decode4 (GMS v92+ only) lands in nPhase (+0x48) —
//     v95 @0x437fde.
//
// The client derives a mist's own lifetime from its WZ skill data
// (tEnd = tStart + 1000*SKILLENTRY::GetLevelData(...), v83 @0x43200f) and does
// not set tEnd at all on the mob-skill arms (130/131, v83 @0x4321cb/0x43206d),
// where removal is driven entirely by AffectedAreaRemoved. No duration is
// carried on this packet.
//
// Legacy GMS divergences (task-165 Tier B, read from disassembly):
//
//	gms_v12 @0x4166f5: dwId(4), nType(1), nSkillID(4), nSLV(1), skillDelay(2),
//	                   rcArea(16) — no dwOwnerId, no nElemAttr.
//	gms_v48 @0x421854: dwId(4), nType(1), dwOwnerId(4), nSkillID(4), nSLV(1),
//	                   skillDelay(2), rcArea(16), nElemAttr(1).
//	gms_v92 @0x4392a0: identical to gms_v95 (nElemAttr + nPhase).
//
// dwId is a uint32 derived from the mist UUID via uuid.UUID.ID() — Atlas uses
// UUIDs internally for mist identity but the protocol carries a 32-bit object id
// for the AFFECTED_AREA_CREATED / AFFECTED_AREA_REMOVED pair.
//
// rcArea is an absolute LTRB rectangle: origin + offset for each corner. origin
// and the lt/rb offsets are constructor inputs only — they are combined into the
// 4×int32 absolute RECT on the wire and are NOT emitted independently.
// packet-audit:fname CAffectedAreaPool::OnAffectedAreaCreated
type AffectedAreaCreated struct {
	mistId     uuid.UUID
	ownerId    uint32
	nType      int32
	skillId    int32
	skillLevel byte
	skillDelay int16
	originX    int16
	originY    int16
	ltX        int16
	ltY        int16
	rbX        int16
	rbY        int16
	elemAttr   int32
	phase      int32
}

func NewAffectedAreaCreated(mistId uuid.UUID, ownerId uint32, nType int32, skillId int32, skillLevel byte, skillDelay int16, originX, originY, ltX, ltY, rbX, rbY int16, elemAttr, phase int32) AffectedAreaCreated {
	return AffectedAreaCreated{
		mistId:     mistId,
		ownerId:    ownerId,
		nType:      nType,
		skillId:    skillId,
		skillLevel: skillLevel,
		skillDelay: skillDelay,
		originX:    originX,
		originY:    originY,
		ltX:        ltX,
		ltY:        ltY,
		rbX:        rbX,
		rbY:        rbY,
		elemAttr:   elemAttr,
		phase:      phase,
	}
}

func (m AffectedAreaCreated) MistId() uuid.UUID { return m.mistId }
func (m AffectedAreaCreated) OwnerId() uint32   { return m.ownerId }
func (m AffectedAreaCreated) NType() int32      { return m.nType }
func (m AffectedAreaCreated) SkillId() int32    { return m.skillId }
func (m AffectedAreaCreated) SkillLevel() byte  { return m.skillLevel }
func (m AffectedAreaCreated) SkillDelay() int16 { return m.skillDelay }
func (m AffectedAreaCreated) OriginX() int16    { return m.originX }
func (m AffectedAreaCreated) OriginY() int16    { return m.originY }
func (m AffectedAreaCreated) LtX() int16        { return m.ltX }
func (m AffectedAreaCreated) LtY() int16        { return m.ltY }
func (m AffectedAreaCreated) RbX() int16        { return m.rbX }
func (m AffectedAreaCreated) RbY() int16        { return m.rbY }
func (m AffectedAreaCreated) ElemAttr() int32   { return m.elemAttr }
func (m AffectedAreaCreated) Phase() int32      { return m.phase }
func (m AffectedAreaCreated) Operation() string { return AffectedAreaCreatedWriter }
func (m AffectedAreaCreated) String() string {
	return fmt.Sprintf("mistId [%s], ownerId [%d], nType [%d], skillId [%d], skillLevel [%d], skillDelay [%d], origin [%d,%d], lt [%d,%d], rb [%d,%d], elemAttr [%d], phase [%d]",
		m.mistId, m.ownerId, m.nType, m.skillId, m.skillLevel, m.skillDelay, m.originX, m.originY, m.ltX, m.ltY, m.rbX, m.rbY, m.elemAttr, m.phase)
}

func (m AffectedAreaCreated) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	// A second trailing Decode4 (nPhase, +0x48) was added after v87 (GMS).
	// GMS v92 CAffectedAreaPool::OnAffectedAreaCreated @0x4392a0 reads two
	// trailing Decode4s (stored at +0x30 nElemAttr and +0x48 nPhase); v61
	// @0x423edc and v83 @0x431ade read exactly one (nElemAttr, +0x30, stored
	// @0x431b3b). v83/v84/v87/JMS185 do not carry nPhase.
	hasPhase := t.IsRegion("GMS") && t.MajorAtLeast(92)
	// nType widened from Decode1 to Decode4 between v48 and v61 (GMS).
	// v12 @0x4166f5 and v48 @0x421854 read Decode1; v61 @0x423edc reads Decode4.
	wideNType := !t.IsRegion("GMS") || t.MajorAtLeast(61)
	// dwOwnerId does not exist before v48 (GMS). v12 @0x4166f5 reads
	// dwId, nType(1), nSkillID(4), nSLV(1) — the 3rd read is compared against the
	// mist skill ids 130/131/2111003 @0x4167cc, proving it is nSkillID, not an owner.
	hasOwnerId := !t.IsRegion("GMS") || t.MajorAtLeast(48)
	// nElemAttr is absent on v12 and is a single byte on v48 (@0x4218c4
	// Decode1) before widening to Decode4 at v61. It always feeds the record's
	// +0x30 slot (v48 @0x421928, v61 @0x423fb0, v83 @0x431b3b, v95 @0x437fd9).
	trailing := trailingWidthWide
	if t.IsRegion("GMS") && !t.MajorAtLeast(48) {
		trailing = trailingWidthNone
	} else if t.IsRegion("GMS") && !t.MajorAtLeast(61) {
		trailing = trailingWidthByte
	}
	return func(options map[string]interface{}) []byte {
		w.WriteInt(mistKey(m.mistId)) // dwId
		if wideNType {
			w.WriteInt32(m.nType)
		} else {
			w.WriteByte(byte(m.nType))
		}
		if hasOwnerId {
			w.WriteInt(m.ownerId) // dwOwnerId
		}
		w.WriteInt32(m.skillId)
		w.WriteByte(m.skillLevel)
		w.WriteInt16(m.skillDelay)
		// rcArea — absolute LTRB RECT (origin + offset), 4×int32.
		w.WriteInt32(int32(m.originX + m.ltX))
		w.WriteInt32(int32(m.originY + m.ltY))
		w.WriteInt32(int32(m.originX + m.rbX))
		w.WriteInt32(int32(m.originY + m.rbY))
		switch trailing {
		case trailingWidthWide:
			w.WriteInt32(m.elemAttr)
		case trailingWidthByte:
			// v48 narrows nElemAttr to one byte (it still feeds +0x30, stored raw
			// @0x421928); the model carries only the 32-bit value, so the low byte
			// is emitted to keep the frame length the client expects.
			w.WriteByte(byte(m.elemAttr))
		case trailingWidthNone:
		}
		if hasPhase {
			w.WriteInt32(m.phase)
		}
		return w.Bytes()
	}
}

// mistKey derives the wire object id (uint32) from a mist UUID. We use
// uuid.UUID.ID() which returns the first 4 bytes (the time_low portion) — Atlas
// uses uuid.UUID for mist identity internally; the wire format requires a
// 32-bit identifier. Collision risk across concurrent mists in a field is
// negligible at mist densities.
func mistKey(id uuid.UUID) uint32 {
	return id.ID()
}
