package serverbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const ItcStatusChargeHandle = "ItcStatusChargeHandle"

// ItcStatusCharge - CITC::OnStatusCharge
// packet-audit:fname CITC::OnStatusCharge
//
// Bodiless (opcode-only) request. The client send function CITC::OnStatusCharge
// (per version: gms_v83 @0x59ebda, gms_v84 @0x5aef76, gms_v87 @0x5ce90b,
// gms_v95 @0x572a50, jms_v185 @0x6040a9) is a uniform shape across all five
// builds:
//
//	if ( !this->m_bITCRequestSent ) {        // "ITC request already sent" latch
//	    this->m_bITCRequestSent = 1;
//	    COutPacket::COutPacket(&pkt, OPCODE); // opcode only — no field writes
//	    SendPacket(..., &pkt);                // sent immediately
//	    ZArray<unsigned char>::RemoveAll(...);
//	}
//
// There are ZERO Encode calls between the COutPacket constructor and
// SendPacket — the request carries no payload (the open-NX-recharge hook). The
// latch guards against a double-send; it does not write to the wire.
type ItcStatusCharge struct{}

func (m ItcStatusCharge) Operation() string {
	return ItcStatusChargeHandle
}

func (m ItcStatusCharge) String() string {
	return ""
}

func (m ItcStatusCharge) Encode(_ logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		return []byte{}
	}
}

func (m *ItcStatusCharge) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
