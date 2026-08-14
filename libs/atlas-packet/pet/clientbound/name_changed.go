package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const PetNameChangedWriter = "PetNameChanged"

// NameTagLayer is the CLife::MakeNameTag decoration selector. GMS v95's
// PDB-backed decompile names it outright (@0x6a125a-0x6a1271):
//
//	if (CInPacket::Decode1(p)) nNameTag = this->m_pTemplate->nNameTag; else nNameTag = 0;
//
// so it selects whether the pet TEMPLATE's decorative name-tag layer is drawn —
// it is NOT a boolean "has a name". v83 (@0x704840), v61 (@0x613615) and v92 do
// the same through unnamed offsets.
//
// The same value already rides the spawn body as Activated.nameTag. The two MUST
// agree: a rename that writes 1 while the spawn writes 0 makes the decoration
// appear on rename and disappear on the next respawn. Atlas has no per-pet
// name-tag inventory, so both write 0. This is a render selector, not a
// per-version wire code, which is why it is a named constant rather than a
// tenant-config lookup (design §5 A5; DOM-25 requires provenance, not a config
// key nobody will tune).
const NameTagLayer = byte(0)

// packet-audit:fname CPet::OnNameChanged
//
// ownerId + slot are the upstream prefix CUser::OnPetPacket reads before
// dispatching to the per-op leaf; every pet codec in this package carries them
// and v61_test.go byte-verifies the framing for the whole family.
// packet-audit:fname CPet::OnNameChanged
type NameChanged struct {
	ownerId uint32
	slot    int8
	name    string
	nameTag byte
}

func NewPetNameChanged(ownerId uint32, slot int8, name string) NameChanged {
	return NameChanged{ownerId: ownerId, slot: slot, name: name, nameTag: NameTagLayer}
}

func (m NameChanged) Operation() string { return PetNameChangedWriter }

func (m NameChanged) String() string {
	return fmt.Sprintf("ownerId [%d], slot [%d], name [%s]", m.ownerId, m.slot, m.name)
}

func (m NameChanged) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		w.WriteInt8(m.slot)
		w.WriteAsciiString(m.name)
		// GMS reads a trailing flag byte; JMS v185 does not. jms
		// CPet::OnNameChanged @0x76a5de performs exactly one DecodeStr and
		// branches on sub_768D82(this) — client state, not the wire.
		if t.IsRegion("GMS") {
			w.WriteByte(m.nameTag)
		}
		return w.Bytes()
	}
}

func (m *NameChanged) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerId = r.ReadUint32()
		m.slot = r.ReadInt8()
		m.name = r.ReadAsciiString()
		if t.IsRegion("GMS") {
			m.nameTag = r.ReadByte()
		}
	}
}
