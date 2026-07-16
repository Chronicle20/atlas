package clientbound

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// TownPortal is the PARTY_OPERATION sub-op that sets (or clears) ONE party
// member's Mystic Door town portal in the client's party town-portal array
// (CWvsContext aTownPortal[slot]). While a viewer is in a party the v83 client
// renders town doors SOLELY from this array (CField::OnTownPortalChanged
// @0x5365c8 ignores the solo SPAWN_PORTAL), so a door cast/removed while in a
// party must update it here — a per-slot surgical update fired on door
// create/remove rather than a full PARTYDATA reload.
//
// Body (per IDA OnPartyResult): Decode1 mode, Decode1 slot, Decode4 townId,
// Decode4 targetId, [Decode4 skillId — GMS v95+ only], Decode2 x, Decode2 y ->
// aTownPortal[slot]. A targetId of EmptyMapId (999999999) clears the slot (the
// render loop skips field ids == 999999999). The mode byte is version-resolved
// via the operations table; the per-version cases (modes shift non-uniformly):
//
//	v83 @0xa3e31c case 0x25      v84 @0xa89cf3 case 0x28
//	v87 @0xad697a case 0x29      v95 @0xa10ab0 case 0x2E (adds skillId)
//	jms @0xb297e7 case 0x28
//
// The GMS v95+ skillId matches the 5-int aTownPortal in WritePartyData (written
// as 0 there); we encode 0 too. JMS uses the small (4-int) layout like v83.
type TownPortal struct {
	mode        byte
	slot        byte
	townMapId   _map.Id
	targetMapId _map.Id
	x           int16
	y           int16
}

func NewTownPortal(mode byte, slot byte, townMapId _map.Id, targetMapId _map.Id, x int16, y int16) TownPortal {
	return TownPortal{mode: mode, slot: slot, townMapId: townMapId, targetMapId: targetMapId, x: x, y: y}
}

// NewTownPortalClear builds a TownPortal that clears the given slot (door
// removed) by encoding the empty-map sentinel in both map ids.
func NewTownPortalClear(mode byte, slot byte) TownPortal {
	return TownPortal{mode: mode, slot: slot, townMapId: _map.EmptyMapId, targetMapId: _map.EmptyMapId}
}

func (m TownPortal) Mode() byte           { return m.mode }
func (m TownPortal) Slot() byte           { return m.slot }
func (m TownPortal) TownMapId() _map.Id   { return m.townMapId }
func (m TownPortal) TargetMapId() _map.Id { return m.targetMapId }
func (m TownPortal) X() int16             { return m.x }
func (m TownPortal) Y() int16             { return m.y }
func (m TownPortal) Operation() string    { return PartyOperationWriter }
func (m TownPortal) String() string {
	return fmt.Sprintf("mode [%d], slot [%d], townMapId [%d], targetMapId [%d], x [%d], y [%d]", m.mode, m.slot, m.townMapId, m.targetMapId, m.x, m.y)
}

// townPortalHasSkillId reports whether this tenant's aTownPortal entry carries
// the 5th m_nSKillID int (GMS v95+). Mirrors WritePartyData's v95plus gate;
// JMS uses the small (4-int) layout like v83.
func townPortalHasSkillId(t tenant.Model) bool {
	return t.Region() == "GMS" && t.MajorVersion() >= 95
}

func (m TownPortal) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.slot)
		w.WriteInt(uint32(m.townMapId))
		w.WriteInt(uint32(m.targetMapId))
		if townPortalHasSkillId(t) {
			w.WriteInt(0) // m_nSKillID (GMS v95+); WritePartyData also writes 0
		}
		w.WriteShort(uint16(m.x))
		w.WriteShort(uint16(m.y))
		return w.Bytes()
	}
}

func (m *TownPortal) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.slot = r.ReadByte()
		m.townMapId = _map.Id(r.ReadUint32())
		m.targetMapId = _map.Id(r.ReadUint32())
		if townPortalHasSkillId(t) {
			_ = r.ReadUint32() // m_nSKillID (GMS v95+)
		}
		m.x = int16(r.ReadUint16())
		m.y = int16(r.ReadUint16())
	}
}
