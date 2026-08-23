package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const DueyActionHandle = "DueyActionHandle"

// Action is the DUEY_ACTION serverbound mode-byte dispatcher. Every client
// send site (CTabSend::SendParcel, CTabReceive::ReceiveParcel,
// CTabReceive::DiscardParcel, CParcelDlg::CloseParcelDlg) leads with this
// single mode byte; docs/packets/dispatchers/duey_action.yaml is the
// mode-resolution source of truth (task-241 Task 6/10).
type Action struct {
	mode byte
}

func (m Action) Mode() byte { return m.mode }

func (m Action) Operation() string {
	return DueyActionHandle
}

func (m Action) String() string {
	return fmt.Sprintf("mode [%d]", m.mode)
}

func (m Action) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *Action) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
