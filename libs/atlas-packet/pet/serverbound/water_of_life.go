package serverbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

const WaterOfLifeHandle = "WaterOfLifeHandle"

// WaterOfLife - CWvsContext::SendWaterOfLife
//
// The body is empty on every applicable version. The client reaches this from
// CWvsContext::SendEtcCashItemUseRequest (gms_v83 @0xa1dc5b), which switches on
// get_etc_cash_item_type (@0x486845) and, for classification 518 => type 5,
// calls SendWaterOfLife() -- a distinct opcode with no body, NOT a CASH_ITEM_USE
// sub-body. Verified per IDB: v83 @0xa1dce6, v84 @0xa68f85, v87 @0xab501c,
// v92 @0x9c6f90, v95 @0x9f28e0 -- each is COutPacket(op) + SendPacket with zero
// Encode* calls. No field diverges, so there are no version gates.
//
// Every operand is derived server-side: the target pet and the consumed Water
// of Life are resolved by atlas-channel, not named on the wire.
// packet-audit:fname CWvsContext::SendWaterOfLife
type WaterOfLife struct{}

func (m WaterOfLife) Operation() string {
	return WaterOfLifeHandle
}

func (m WaterOfLife) String() string {
	return ""
}

func (m WaterOfLife) Encode(_ logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		return []byte{}
	}
}

func (m *WaterOfLife) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
