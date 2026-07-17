package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const OwlActionHandle = "OwlActionHandle"

// packet-audit:fname CUIShopScanner::OnCreate
// OwlAction is sent once when the shop-scanner UI opens (mode 5) to request
// the most-searched hot list. A full construction-site scan of every
// COutPacket(0x42) (v83) / COutPacket(0x48) (v95) found exactly one sender:
// CUIShopScanner::OnCreate with mode 5 (task-127 design §1.3).
type OwlAction struct {
	mode byte
}

func NewOwlAction(mode byte) OwlAction {
	return OwlAction{mode: mode}
}

func (m OwlAction) Mode() byte {
	return m.mode
}

func (m OwlAction) Operation() string {
	return OwlActionHandle
}

func (m OwlAction) String() string {
	return fmt.Sprintf("mode [%d]", m.mode)
}

func (m OwlAction) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *OwlAction) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
