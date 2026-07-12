package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MtsChargeParamResultWriter = "MtsChargeParamResult"

// MtsChargeParamResult is the bodiless "charge parameter result" the client
// expects in response to ITC_STATUS_CHARGE (the MTS "Charge" button). The v83
// client's CITC::OnChargeParamResult (IDA 0x5a4241, dispatched by CITC::OnPacket
// case 346) reads NOTHING from the packet: it clears the request latch
// (this[6]=0) and opens the cash-charge web page. The opcode alone is the signal,
// so the payload is empty.
//
// The clientbound opcode is version-specific (= MtsOperation2 - 1, IDA-verified
// from each client's CITC::OnPacket dispatch): v83 0x15A, v84 0x164, v87 0x16F,
// v95 0x19A. It is resolved from the tenant socket.writers config by writer name.
//
// packet-audit:fname CITC::OnChargeParamResult
type MtsChargeParamResult struct{}

func NewMtsChargeParamResult() MtsChargeParamResult { return MtsChargeParamResult{} }

func (m MtsChargeParamResult) Operation() string { return MtsChargeParamResultWriter }
func (m MtsChargeParamResult) String() string    { return "charge param result" }

func (m MtsChargeParamResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *MtsChargeParamResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {}
}
