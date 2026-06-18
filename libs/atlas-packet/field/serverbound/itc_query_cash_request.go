package serverbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const ItcQueryCashRequestHandle = "ItcQueryCashRequestHandle"

// ItcQueryCashRequest - CITC::TrySendQueryCashRequest
// packet-audit:fname CITC::TrySendQueryCashRequest
//
// Bodiless (opcode-only) request. The client send function
// CITC::TrySendQueryCashRequest (per version: gms_v83 @0x59eece, gms_v84
// @0x5af26a, gms_v87 @0x5cec92, gms_v95 @0x572ad0, jms_v185 @0x6043bf) is a
// uniform shape across all five builds:
//
//	if ( this->m_bITCRequestSent )           // "ITC request already sent" latch
//	    return 0;                            // (this[6] in the unnamed builds)
//	COutPacket::COutPacket(&pkt, OPCODE);    // opcode only — no field writes
//	SendPacket(..., &pkt);                   // sent immediately
//	this->m_bITCRequestSent = 1;
//	ZArray<unsigned char>::RemoveAll(...);
//	return 1;
//
// There are ZERO Encode calls between the COutPacket constructor and
// SendPacket — the request carries no payload. It is the wallet-balance query
// that elicits the clientbound MTS_OPERATION2 (CITC::OnQueryCashResult). The
// latch guards against a double-send; it does not write to the wire.
type ItcQueryCashRequest struct{}

func (m ItcQueryCashRequest) Operation() string {
	return ItcQueryCashRequestHandle
}

func (m ItcQueryCashRequest) String() string {
	return ""
}

func (m ItcQueryCashRequest) Encode(_ logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		return []byte{}
	}
}

func (m *ItcQueryCashRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
