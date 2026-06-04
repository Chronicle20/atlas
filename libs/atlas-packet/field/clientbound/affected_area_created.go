package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const AffectedAreaCreatedWriter = "AffectedAreaCreated"

// AffectedAreaCreated is the clientbound packet that announces a new
// affected-area (mist) on the field. The wire body follows the client read-order
// in CAffectedAreaPool::OnAffectedAreaCreated:
//
//	Decode4 dwId, Decode4 nType, Decode4 dwOwnerId, Decode4 nSkillID,
//	Decode1 nSLV, Decode2 phase, DecodeBuf(16) rcArea (4×int32 absolute RECT),
//	[Decode4 tStart — v95 GMS only], Decode4 tEnd.
//
// dwId is a uint32 derived from the mist UUID via uuid.UUID.ID() — Atlas uses
// UUIDs internally for mist identity but the protocol carries a 32-bit object id
// for the AFFECTED_AREA_CREATED / AFFECTED_AREA_REMOVED pair.
//
// rcArea is an absolute LTRB rectangle: origin + offset for each corner. origin
// and the lt/rb offsets are constructor inputs only — they are combined into the
// 4×int32 absolute RECT on the wire and are NOT emitted independently.
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
	// tStart was inserted between rcArea and tEnd in v95 (GMS). v83/v87/JMS185
	// do not carry it.
	v95Plus := t.Region() == "GMS" && t.MajorVersion() >= 95
	return func(options map[string]interface{}) []byte {
		w.WriteInt(mistKey(m.mistId)) // dwId
		w.WriteInt32(m.nType)
		w.WriteInt(m.ownerId) // dwOwnerId
		w.WriteInt32(m.skillId)
		w.WriteByte(m.skillLevel)
		w.WriteInt16(m.phase)
		// rcArea — absolute LTRB RECT (origin + offset), 4×int32.
		w.WriteInt32(int32(m.originX + m.ltX))
		w.WriteInt32(int32(m.originY + m.ltY))
		w.WriteInt32(int32(m.originX + m.rbX))
		w.WriteInt32(int32(m.originY + m.rbY))
		if v95Plus {
			w.WriteInt32(m.tStart)
		}
		w.WriteInt32(m.tEnd)
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
