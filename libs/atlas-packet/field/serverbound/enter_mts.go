package serverbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

const EnterMtsHandle = "EnterMtsHandle"

// EnterMts - CWvsContext::SendMigrateToITCRequest
// packet-audit:fname CWvsContext::SendMigrateToITCRequest
//
// Bodiless (opcode-only) request. The client send site at 0xa1263b
// (CWvsContext::SendMigrateToITCRequest @0xa12522) constructs
// COutPacket(opcode 0x9C) and immediately SendPacket()s it with ZERO Encode
// calls in between. All preceding code in the sender (guest-ID guard,
// lie-detector guard, map-flag guard) emits local chat/dialog and returns
// early; none writes to the packet. The request therefore carries no payload.
// packet-audit:fname CWvsContext::SendMigrateToITCRequest
type EnterMts struct{}

func (m EnterMts) Operation() string {
	return EnterMtsHandle
}

func (m EnterMts) String() string {
	return ""
}

func (m EnterMts) Encode(_ logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		return []byte{}
	}
}

func (m *EnterMts) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
