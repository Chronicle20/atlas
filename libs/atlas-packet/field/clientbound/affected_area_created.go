package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const AffectedAreaCreatedWriter = "AffectedAreaCreated"

// AffectedAreaCreated is the v83 clientbound packet that announces a new
// affected-area (mist) on the field. The wire body uses a uint32 mist key,
// derived from the first 4 bytes of the UUID via uuid.UUID.ID() — Atlas uses
// UUIDs internally for mist identity but the v83 protocol carries a 32-bit
// object id for the AFFECTED_AREA_CREATED / AFFECTED_AREA_REMOVED pair.
type AffectedAreaCreated struct {
	mistId     uuid.UUID
	ownerId    uint32
	originX    int16
	originY    int16
	ltX        int16
	ltY        int16
	rbX        int16
	rbY        int16
	duration   int64
	skillLevel uint32
}

func NewAffectedAreaCreated(mistId uuid.UUID, ownerId uint32, originX, originY, ltX, ltY, rbX, rbY int16, duration int64, skillLevel uint32) AffectedAreaCreated {
	return AffectedAreaCreated{
		mistId:     mistId,
		ownerId:    ownerId,
		originX:    originX,
		originY:    originY,
		ltX:        ltX,
		ltY:        ltY,
		rbX:        rbX,
		rbY:        rbY,
		duration:   duration,
		skillLevel: skillLevel,
	}
}

func (m AffectedAreaCreated) MistId() uuid.UUID  { return m.mistId }
func (m AffectedAreaCreated) OwnerId() uint32    { return m.ownerId }
func (m AffectedAreaCreated) OriginX() int16     { return m.originX }
func (m AffectedAreaCreated) OriginY() int16     { return m.originY }
func (m AffectedAreaCreated) LtX() int16         { return m.ltX }
func (m AffectedAreaCreated) LtY() int16         { return m.ltY }
func (m AffectedAreaCreated) RbX() int16         { return m.rbX }
func (m AffectedAreaCreated) RbY() int16         { return m.rbY }
func (m AffectedAreaCreated) Duration() int64    { return m.duration }
func (m AffectedAreaCreated) SkillLevel() uint32 { return m.skillLevel }
func (m AffectedAreaCreated) Operation() string  { return AffectedAreaCreatedWriter }
func (m AffectedAreaCreated) String() string {
	return fmt.Sprintf("mistId [%s], ownerId [%d], origin [%d,%d], lt [%d,%d], rb [%d,%d], duration [%d], skillLevel [%d]",
		m.mistId, m.ownerId, m.originX, m.originY, m.ltX, m.ltY, m.rbX, m.rbY, m.duration, m.skillLevel)
}

func (m AffectedAreaCreated) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(mistKey(m.mistId))
		w.WriteInt(m.ownerId)
		w.WriteInt16(m.originX)
		w.WriteInt16(m.originY)
		w.WriteInt16(m.ltX)
		w.WriteInt16(m.ltY)
		w.WriteInt16(m.rbX)
		w.WriteInt16(m.rbY)
		w.WriteInt32(int32(m.duration))
		w.WriteInt(m.skillLevel)
		return w.Bytes()
	}
}

// mistKey derives the v83 wire object id (uint32) from a mist UUID. We use
// uuid.UUID.ID() which returns the first 4 bytes (the time_low portion) — Atlas
// uses uuid.UUID for mist identity internally; the wire format requires a
// 32-bit identifier. Collision risk across concurrent mists in a field is
// negligible at v83 mist densities.
func mistKey(id uuid.UUID) uint32 {
	return id.ID()
}
