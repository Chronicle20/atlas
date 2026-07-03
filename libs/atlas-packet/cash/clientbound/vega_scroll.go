package clientbound

import (
	"context"
	"fmt"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// VegaScroll — CUIVega::OnVegaResult (v83 0x82d8d5 via OnPacket 0x82d8bf
// opcode 0x166; v95 0x7bf7b0 via OnPacket 0x7c0680 opcode 0x1AD). Body is a
// single mode byte. The accepted values are version-shifted (+4 from v83 to
// v95) AND v95 selects its result popup from the START byte while v83 renders
// EffectSuccess/EffectFail from the RESULT byte — so the operations keys are
// outcome-keyed and the byte is resolved from the tenant operations table at
// encode time (task-130 design §2.2–§2.3). On v83 both START keys collapse to
// 0x40 harmlessly. Any unconfigured key resolves to 99, which both clients
// route to the safe "This item cannot be used." notice arm (no crash arm
// exists in either version).
const VegaScrollWriter = "VegaScroll"

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
// resolution, no 3s timer), so the start byte can be outcome-selected; v83
// collapses both keys to the same byte.
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
