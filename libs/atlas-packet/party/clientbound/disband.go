package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// packet-audit:fname CWvsContext::OnPartyResult#Disband
type Disband struct {
	mode     byte
	partyId  uint32
	targetId uint32
}

func NewDisband(mode byte, partyId uint32, targetId uint32) Disband {
	return Disband{mode: mode, partyId: partyId, targetId: targetId}
}

func (m Disband) Mode() byte       { return m.mode }
func (m Disband) PartyId() uint32  { return m.partyId }
func (m Disband) TargetId() uint32 { return m.targetId }

func (m Disband) Operation() string {
	return PartyOperationWriter
}

func (m Disband) String() string {
	return fmt.Sprintf("mode [%d], partyId [%d], targetId [%d]", m.mode, m.partyId, m.targetId)
}

func (m Disband) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	// GMS legacy (< v61): the else-branch of case 11 stops after Decode1(=0) — it
	// does NOT append the repeated partyId that v61/v83+ emit (IDA v48
	// OnPartyResult@0x729935 case-11 else). task-113 v48 close-I. v28 is
	// unverified-by-inference (no v28 IDB) — folded into the v48 legacy shape.
	legacyNoTrailer := t.IsRegion("GMS") && t.MajorVersion() < 61
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.partyId)
		w.WriteInt(m.targetId)
		w.WriteByte(0)
		if !legacyNoTrailer {
			w.WriteInt(m.partyId)
		}
		return w.Bytes()
	}
}

func (m *Disband) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	// GMS legacy (< v61): no trailing repeated partyId (see Encode). task-113 close-I.
	legacyNoTrailer := t.IsRegion("GMS") && t.MajorVersion() < 61
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.partyId = r.ReadUint32()
		m.targetId = r.ReadUint32()
		_ = r.ReadByte() // constant 0
		if !legacyNoTrailer {
			_ = r.ReadUint32() // partyId repeated
		}
	}
}
