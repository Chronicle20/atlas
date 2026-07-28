package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// RPSActionHandle is the config key for the CRPSGameDlg RPS_ACTION serverbound
// dispatcher. Six senders share ONE opcode (RPS_ACTION) with a leading sub-op
// byte: OnBtStart=0, SendSelection=1 (+throw body), Update(timeout)=2,
// OnBtContinue=3, OnBtExit=4, OnBtRetry=5 — IDENTICAL across all five versions
// per docs/tasks/task-132-rps-npc-game/ida-rps-serverbound.md §0/§6 (only the
// opcode shifts). Five of the six arms (all but SendSelection) are bodyless —
// the sub-op byte alone IS the full wire content, so this generic Operation
// decodes exactly that.
const RPSActionHandle = "RPSActionHandle"

// Operation - CRPSGameDlg (bodyless arms: OnBtStart/OnBtContinue/OnBtExit/
// OnBtRetry/Update). Only SendSelection (mode 1) carries a body beyond the
// mode byte — see OperationSelect.
// packet-audit:fname CRPSGameDlg::OnBtStart
type Operation struct {
	mode byte
}

func (m Operation) Mode() byte { return m.mode }

func (m Operation) Operation() string {
	return RPSActionHandle
}

func (m Operation) String() string {
	return fmt.Sprintf("mode [%d]", m.mode)
}

func (m Operation) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *Operation) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
