package serverbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CharacterStateChangeByPortableChairHandle = "CharacterStateChangeByPortableChairHandle"

// StateChangeByPortableChair - CWvsContext::TryRecovery
// (STATE_CHANGE_BY_PORTABLE_CHAIR_REQUEST, STATUS row 562).
//
// The body is EMPTY in every supported version: the client constructs
// COutPacket(ctor, opcode) and calls CClientSocket::SendPacket with zero
// Encode calls in between (IDA-verified, task-141 design §2.1; the four legacy
// columns IDA-verified during the task-141 merge — see the tail of
// CWvsContext::TryRecovery: IsTherePortableChairStatUp → SetPortableChairStatSetSent
// → COutPacket(opcode)/SendPacket, no Encode):
//
//	gms_v48  CWvsContext::TryRecovery @ 0x70a977, opcode 0x3A
//	gms_v61  CWvsContext::TryRecovery @ 0x82ceaa, opcode 0x45
//	gms_v72  CWvsContext::TryRecovery @ 0x8fe5c1, opcode 0x49
//	gms_v79  CWvsContext::TryRecovery @ 0x94f24a, opcode 0x48
//	gms_v83  CWvsContext::TryRecovery @ 0xa02e34, send site 0xa032ad, opcode 0x4A
//	gms_v84  sub_A4D05A               @ 0xa4d05a (structurally identical), opcode 0x4A
//	gms_v87  CWvsContext::TryRecovery @ 0xa97e50, opcode 0x4D
//	gms_v95  CWvsContext::TryRecovery @ 0x9d4020, opcode 0x50
//	jms_v185 CWvsContext::TryRecovery @ 0xae6f5a, opcode 0x42
//
// Send gate (identical semantics in all nine): CanSendExclRequest(500, 0)
// passes, an active portable chair id is set, time since sitting >= 20000 ms,
// and a per-sit latch is unset — so the packet fires AT MOST ONCE PER SIT,
// and only for portable chairs whose item data has no `spec` node. No
// clientbound response exists; the client latches locally. Chair recovery
// amounts do NOT ride this packet — they ride HEAL_OVER_TIME (row 577).
type StateChangeByPortableChair struct{}

func (m StateChangeByPortableChair) Operation() string {
	return CharacterStateChangeByPortableChairHandle
}

func (m StateChangeByPortableChair) String() string {
	return "state change by portable chair (empty body)"
}

func (m StateChangeByPortableChair) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *StateChangeByPortableChair) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
