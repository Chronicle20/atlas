package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const PetSpawnHandle = "PetSpawnHandle"

// packet-audit:fname CWvsContext::SendActivatePetRequest
type Spawn struct {
	updateTime uint32
	slot       int16
	lead       bool
}

func (m Spawn) UpdateTime() uint32 {
	return m.updateTime
}

func (m Spawn) Slot() int16 {
	return m.slot
}

func (m Spawn) Lead() bool {
	return m.lead
}

func (m Spawn) Operation() string {
	return PetSpawnHandle
}

func (m Spawn) String() string {
	return fmt.Sprintf("updateTime [%d] slot [%d] lead [%t]", m.updateTime, m.slot, m.lead)
}

// hasPetSpawnLead reports whether the pet-activate request carries the trailing
// "lead" byte.
//
// VERIFIED BOUNDS: absent on v48, present on v72. IDA v48
// CWvsContext::SendActivatePetRequest @0x71d118 builds COutPacket(77) and encodes
// only Encode4(tick) then Encode2(slot) before SendPacket. v72 @0x91c241 builds
// COutPacket(97) and encodes Encode4(tick) @0x91c513, Encode2(slot) @0x91c51e AND
// Encode1(lead) @0x91c529 - the byte is the "swap out the current pet?" answer
// gathered from CUtilDlg::YesNo just above. v87 @0xabbb70 and v95 @0x9f6980 keep
// the three-field shape.
//
// v61 is INFERRED, not verified: the v61 IDB has no named equivalent (only
// SendPetFoodItemUseRequest @0x831de9 is named) and gms_v61.yaml registers no
// serverbound SPAWN_PET at all, so no matrix cell depends on this choice. The
// boundary is placed at 61 to match every other field delta found on this branch;
// if a v61 tenant ever routes this op, re-derive the send-site before trusting it.
func hasPetSpawnLead(t tenant.Model) bool {
	return (t.IsRegion("GMS") && t.MajorAtLeast(61)) || t.Region() == "JMS"
}

func (m Spawn) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.slot)
		if hasPetSpawnLead(t) {
			w.WriteBool(m.lead)
		}
		return w.Bytes()
	}
}

func (m *Spawn) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.slot = r.ReadInt16()
		if hasPetSpawnLead(t) {
			m.lead = r.ReadBool()
		}
	}
}
