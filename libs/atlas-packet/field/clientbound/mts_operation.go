package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MtsOperationWriter = "MtsOperation"

// MtsOperation models CITC::OnNormalItemResult — the MTS / cash-shop "normal
// item" result dispatcher. The client reads a single leading mode byte
// (CInPacket::Decode1) and switch-dispatches to one of 35 sub-handlers
// (0x15..0x3E). Only the leading mode byte is modeled here; the per-mode tail
// is mode-opaque (each arm reads its own structure). This is the OP-MODE-PREFIX
// shape: the wire contract this codec owns is the mode byte that selects the arm.
// packet-audit:fname CITC::OnNormalItemResult#Mode  (dispatcher family — see docs/packets/evidence/families.yaml)
type MtsOperation struct {
	mode byte
}

func NewMtsOperation(mode byte) MtsOperation {
	return MtsOperation{mode: mode}
}

func (m MtsOperation) Mode() byte { return m.mode }

func (m MtsOperation) Operation() string { return MtsOperationWriter }
func (m MtsOperation) String() string {
	return fmt.Sprintf("mts operation mode [%d]", m.mode)
}

func (m MtsOperation) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *MtsOperation) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
