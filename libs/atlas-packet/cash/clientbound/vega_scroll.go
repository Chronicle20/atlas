package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// VegaScroll — CUIVega::OnVegaResult. Body is a single mode byte. The client
// dispatches on the opcode (CUIVega::OnPacket) then reads one Decode1 and routes
// it to three arms: START (twinkle animation + gauge), RESULT (validated,
// latched), or the else-arm ("This item cannot be used." notice). IDA-verified
// per version (task-130 Task 4):
//
//	version   opcode   OnVegaResult   START(succ/fail)  RESULT(succ/fail)  INVALID
//	gms_v83   0x166    0x82d8d5       0x40 / 0x45       0x41 / 0x43        0x42
//	gms_v87   0x17B    0x8919b6       0x42 / 0x47       0x43 / 0x45        0x44
//	gms_v95   0x1AD    0x7bf7b0       0x44 / 0x49       0x45 / 0x47        0x42
//	jms_v185  0x183    0x8b89ad       0x3B / 0x40       0x3C / 0x3E        0x3D
//
// On EVERY version the success/fail popup (SuccessWnd/FailWnd, EffectSuccess/
// EffectFail) is selected by the START byte in CUIVega::Draw — the RESULT byte
// is only range-validated, not used to pick the outcome. So the START byte
// carries the outcome on all versions (the earlier "v83 collapses both START
// keys to 0x40" hypothesis was WRONG — v83 START_FAILURE is 0x45, not 0x40:
// sending 0x40 on a failure would show the SUCCESS window). The values are
// version-shifted (no uniform delta: v87 is v83+2, v95 is v83+4, jms is its
// own map), so the byte is resolved from the tenant operations table at encode
// time under outcome-keyed names. Any unconfigured key resolves to 99, which
// every client routes to the safe notice arm (no crash arm exists).
const VegaScrollWriter = "VegaScroll"

// VegaScroll is the single-byte audit representative for the packet family
// (packet-audit locates `type VegaScroll struct` and compares its 1-byte Encode
// against the client's single Decode1). The production writers are the three
// discrete arm structs below (VegaScrollStart/Result/Invalid); this struct is
// the uncalled audit codec that stands in for all three identically-shaped arms
// (VERIFYING_A_PACKET §9 "uncalled audit codec").
//
// packet-audit:fname CUIVega::OnVegaResult
type VegaScroll struct {
	mode byte
}

func NewVegaScroll(mode byte) VegaScroll { return VegaScroll{mode: mode} }

func (m VegaScroll) Mode() byte        { return m.mode }
func (m VegaScroll) Operation() string { return VegaScrollWriter }
func (m VegaScroll) String() string    { return fmt.Sprintf("vega scroll mode [%d]", m.mode) }

func (m VegaScroll) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *VegaScroll) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

const (
	VegaScrollModeStartSuccess  = "START_SUCCESS"
	VegaScrollModeStartFailure  = "START_FAILURE"
	VegaScrollModeResultSuccess = "RESULT_SUCCESS"
	VegaScrollModeResultFailure = "RESULT_FAILURE"
	VegaScrollModeInvalid       = "INVALID"
)

// VegaScrollStart — the start-animation arm (twinkle sound + gauge).
//
// packet-audit:fname CUIVega::OnVegaResult#Start
type VegaScrollStart struct {
	mode byte
}

func NewVegaScrollStart(mode byte) VegaScrollStart { return VegaScrollStart{mode: mode} }

func (m VegaScrollStart) Mode() byte        { return m.mode }
func (m VegaScrollStart) Operation() string { return VegaScrollWriter }
func (m VegaScrollStart) String() string    { return fmt.Sprintf("vega scroll start mode [%d]", m.mode) }

func (m VegaScrollStart) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *VegaScrollStart) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// VegaScrollResult — the latched pass/fail arm, displayed after the
// animation completes on the client's own clock.
//
// packet-audit:fname CUIVega::OnVegaResult#Result
type VegaScrollResult struct {
	mode byte
}

func NewVegaScrollResult(mode byte) VegaScrollResult { return VegaScrollResult{mode: mode} }

func (m VegaScrollResult) Mode() byte        { return m.mode }
func (m VegaScrollResult) Operation() string { return VegaScrollWriter }
func (m VegaScrollResult) String() string    { return fmt.Sprintf("vega scroll result mode [%d]", m.mode) }

func (m VegaScrollResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *VegaScrollResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// VegaScrollInvalid — the else-arm: the client shows "This item cannot be
// used." and closes the dialog. REQUIRED on rejection (not optional): after
// sending the request the client sets m_bRequestSent and disables the dialog;
// a rejection that sent nothing would leave it wedged (design §2.3).
//
// packet-audit:fname CUIVega::OnVegaResult#Invalid
type VegaScrollInvalid struct {
	mode byte
}

func NewVegaScrollInvalid(mode byte) VegaScrollInvalid { return VegaScrollInvalid{mode: mode} }

func (m VegaScrollInvalid) Mode() byte        { return m.mode }
func (m VegaScrollInvalid) Operation() string { return VegaScrollWriter }
func (m VegaScrollInvalid) String() string {
	return fmt.Sprintf("vega scroll invalid mode [%d]", m.mode)
}

func (m VegaScrollInvalid) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *VegaScrollInvalid) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// VegaScrollStartBody resolves the outcome-keyed START mode from the tenant
// operations table. The server resolves the outcome before sending (immediate
// resolution, no 3s timer), so the start byte can be outcome-selected; on
// every version, including v83, the START byte carries the outcome, since
// CUIVega::Draw picks the success/fail popup from it.
func VegaScrollStartBody(success bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	if success {
		return atlas_packet.WithResolvedCode("operations", VegaScrollModeStartSuccess, func(mode byte) packet.Encoder {
			return NewVegaScrollStart(mode)
		})
	}
	return atlas_packet.WithResolvedCode("operations", VegaScrollModeStartFailure, func(mode byte) packet.Encoder {
		return NewVegaScrollStart(mode)
	})
}

func VegaScrollResultBody(success bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	if success {
		return atlas_packet.WithResolvedCode("operations", VegaScrollModeResultSuccess, func(mode byte) packet.Encoder {
			return NewVegaScrollResult(mode)
		})
	}
	return atlas_packet.WithResolvedCode("operations", VegaScrollModeResultFailure, func(mode byte) packet.Encoder {
		return NewVegaScrollResult(mode)
	})
}

func VegaScrollInvalidBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", VegaScrollModeInvalid, func(mode byte) packet.Encoder {
		return NewVegaScrollInvalid(mode)
	})
}
