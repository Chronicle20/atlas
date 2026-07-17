package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// packet-audit:fname CRPSGameDlg::SendSelection
//
// OperationSelect is the ONLY RPS_ACTION arm with a body: sub-op 1 (SELECT)
// followed by a single RAW throw byte. Per
// docs/tasks/task-132-rps-npc-game/ida-rps-serverbound.md §0/§1-§5,
// OnButtonClicked maps R/P/S button ids 0x7D0/0x7D1/0x7D2 to
// SendSelection(this, nId-2000), and SendSelection does Encode1(1) (mode,
// captured by Operation, not here) then Encode1(throw) with that 0/1/2 value
// UNMODIFIED — pass it through raw, no remap (Rock=0/Paper=1/Scissors=2).
// packet-audit:fname CRPSGameDlg::SendSelection
type OperationSelect struct {
	throw byte
}

func (m OperationSelect) Throw() byte { return m.throw }

func (m OperationSelect) Operation() string {
	return "OperationSelect"
}

func (m OperationSelect) String() string {
	return fmt.Sprintf("throw [%d]", m.throw)
}

func (m OperationSelect) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.throw)
		return w.Bytes()
	}
}

func (m *OperationSelect) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.throw = r.ReadByte()
	}
}
