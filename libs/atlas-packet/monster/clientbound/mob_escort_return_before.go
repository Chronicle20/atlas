package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MobEscortReturnBeforeWriter = "MobEscortReturnBefore"

// MobEscortReturnBefore is the clientbound MOB_ESCORT_RETURN_BEFORE packet
// (CMob::OnEscortReturnBefore): during an escort sequence the server tells the
// client a mob is returning to an earlier waypoint index; the client shows the
// "return" balloon and stores the index.
//
// Byte layout (IDA-verified, a single Decode4):
//   - index : int32 — Decode4; the escort waypoint index to return before
//
// IDA basis: CMob::OnEscortReturnBefore — v95 @0x649410, jms @0x6f029c
// (`v3 = Decode4(iPaket); if (IsActive && index in range) { ...balloon...;
// m_pvcActive escort-return fields = index }`). v95/jms only — the escort family is
// absent from v83/v84/v87 (their dispatchers have no escort cases and no Escort
// symbols).
//
// packet-audit:fname CMob::OnEscortReturnBefore
type MobEscortReturnBefore struct {
	index int32
}

func NewMobEscortReturnBefore(index int32) MobEscortReturnBefore {
	return MobEscortReturnBefore{index: index}
}

func (m MobEscortReturnBefore) Index() int32      { return m.index }
func (m MobEscortReturnBefore) Operation() string { return MobEscortReturnBeforeWriter }
func (m MobEscortReturnBefore) String() string {
	return fmt.Sprintf("index [%d]", m.index)
}

func (m MobEscortReturnBefore) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.index)
		return w.Bytes()
	}
}

func (m *MobEscortReturnBefore) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.index = r.ReadInt32()
	}
}
