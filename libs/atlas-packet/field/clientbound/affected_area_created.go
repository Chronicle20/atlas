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

// trailingWidth selects how the last time word of the body is written. Which
// client slot that word feeds depends on the twoTimeWords gate: when
// twoTimeWords is false it is the only time word and lands in the
// affected-area record's +0x30 slot; when twoTimeWords is true the preceding
// tStart write takes +0x30 (v92 @0x4393b9) and this trailing tEnd write lands
// in +0x48 instead (v92 @0x4393be). See the version gates in
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
//	Decode1 nSLV, Decode2 phase, DecodeBuf(16) rcArea (4×int32 absolute RECT),
//	[Decode4 tStart — GMS v92+], Decode4 tEnd.
//
// Legacy GMS divergences (task-165 Tier B, read from disassembly):
//
//	gms_v12 @0x4166f5: dwId(4), nType(1), nSkillID(4), nSLV(1), phase(2),
//	                   rcArea(16) — no dwOwnerId, no trailing time word.
//	gms_v48 @0x421854: dwId(4), nType(1), dwOwnerId(4), nSkillID(4), nSLV(1),
//	                   phase(2), rcArea(16), trailing(1).
//	gms_v92 @0x4392a0: identical to gms_v95 (two trailing Decode4s).
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
	phase      int16
	originX    int16
	originY    int16
	ltX        int16
	ltY        int16
	rbX        int16
	rbY        int16
	tStart     int32
	tEnd       int32
}

func NewAffectedAreaCreated(mistId uuid.UUID, ownerId uint32, nType int32, skillId int32, skillLevel byte, phase int16, originX, originY, ltX, ltY, rbX, rbY int16, tStart, tEnd int32) AffectedAreaCreated {
	return AffectedAreaCreated{
		mistId:     mistId,
		ownerId:    ownerId,
		nType:      nType,
		skillId:    skillId,
		skillLevel: skillLevel,
		phase:      phase,
		originX:    originX,
		originY:    originY,
		ltX:        ltX,
		ltY:        ltY,
		rbX:        rbX,
		rbY:        rbY,
		tStart:     tStart,
		tEnd:       tEnd,
	}
}

func (m AffectedAreaCreated) MistId() uuid.UUID { return m.mistId }
func (m AffectedAreaCreated) OwnerId() uint32   { return m.ownerId }
func (m AffectedAreaCreated) NType() int32      { return m.nType }
func (m AffectedAreaCreated) SkillId() int32    { return m.skillId }
func (m AffectedAreaCreated) SkillLevel() byte  { return m.skillLevel }
func (m AffectedAreaCreated) Phase() int16      { return m.phase }
func (m AffectedAreaCreated) OriginX() int16    { return m.originX }
func (m AffectedAreaCreated) OriginY() int16    { return m.originY }
func (m AffectedAreaCreated) LtX() int16        { return m.ltX }
func (m AffectedAreaCreated) LtY() int16        { return m.ltY }
func (m AffectedAreaCreated) RbX() int16        { return m.rbX }
func (m AffectedAreaCreated) RbY() int16        { return m.rbY }
func (m AffectedAreaCreated) TStart() int32     { return m.tStart }
func (m AffectedAreaCreated) TEnd() int32       { return m.tEnd }
func (m AffectedAreaCreated) Operation() string { return AffectedAreaCreatedWriter }
func (m AffectedAreaCreated) String() string {
	return fmt.Sprintf("mistId [%s], ownerId [%d], nType [%d], skillId [%d], skillLevel [%d], phase [%d], origin [%d,%d], lt [%d,%d], rb [%d,%d], tStart [%d], tEnd [%d]",
		m.mistId, m.ownerId, m.nType, m.skillId, m.skillLevel, m.phase, m.originX, m.originY, m.ltX, m.ltY, m.rbX, m.rbY, m.tStart, m.tEnd)
}

func (m AffectedAreaCreated) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	// The extra leading time word (written here as tStart) was inserted between
	// rcArea and the legacy trailing time field between v87 and v92 (GMS).
	// GMS v92 CAffectedAreaPool::OnAffectedAreaCreated @0x4392a0 reads two
	// trailing Decode4s (stored at +0x30 and +0x48); v61 @0x423edc and v83 read
	// exactly one (the +0x30 slot). v83/v84/v87/JMS185 do not carry it.
	twoTimeWords := t.IsRegion("GMS") && t.MajorAtLeast(92)
	// nType widened from Decode1 to Decode4 between v48 and v61 (GMS).
	// v12 @0x4166f5 and v48 @0x421854 read Decode1; v61 @0x423edc reads Decode4.
	wideNType := !t.IsRegion("GMS") || t.MajorAtLeast(61)
	// dwOwnerId does not exist before v48 (GMS). v12 @0x4166f5 reads
	// dwId, nType(1), nSkillID(4), nSLV(1) — the 3rd read is compared against the
	// mist skill ids 130/131/2111003 @0x4167cc, proving it is nSkillID, not an owner.
	hasOwnerId := !t.IsRegion("GMS") || t.MajorAtLeast(48)
	// The trailing time word is absent on v12 and is a single byte on v48
	// (@0x4218c4 Decode1) before widening to Decode4 at v61. On the single-word
	// versions it feeds the record's +0x30 slot (v48 @0x421928, v61 @0x423fb0);
	// on v92 it feeds +0x48 (@0x4393be) because tStart has taken +0x30.
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
		w.WriteInt16(m.phase)
		// rcArea — absolute LTRB RECT (origin + offset), 4×int32.
		w.WriteInt32(int32(m.originX + m.ltX))
		w.WriteInt32(int32(m.originY + m.ltY))
		w.WriteInt32(int32(m.originX + m.rbX))
		w.WriteInt32(int32(m.originY + m.rbY))
		if twoTimeWords {
			w.WriteInt32(m.tStart)
		}
		switch trailing {
		case trailingWidthWide:
			w.WriteInt32(m.tEnd)
		case trailingWidthByte:
			// v48 narrows this word to one byte (it still feeds +0x30, stored raw
			// @0x421928); the model carries only the 32-bit value, so the low byte
			// is emitted to keep the frame length the client expects.
			w.WriteByte(byte(m.tEnd))
		case trailingWidthNone:
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
