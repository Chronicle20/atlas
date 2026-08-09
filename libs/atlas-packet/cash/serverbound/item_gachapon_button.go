package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CashItemGachaponHandle = "CashItemGachaponHandle"

// CashItemGachaponButton - the Cash Shop Surprise "Open" button.
// CUICashItemGachapon::OnButtonClicked(nId == 2000) emits
// COutPacket(<send opcode>) + EncodeBuffer(&m_liItemSN, 8) and nothing else.
// EncodeBuffer of a LARGE_INTEGER is byte-identical to a little-endian
// int64, so the body needs no version gate: v79 0x9F, v83 0xA1, v84 0xA5,
// v87 0xA9, v92 0xB6, v95 0xB9, jms_v185 0xA7 all carry the same 8 bytes.
// The client self-gates re-clicks with `if (m_nState < 1)`; only v79 also
// calls CWvsContext::SetExclRequestSent, so on every version in scope the
// send does NOT arm the excl-request gate and no EnableActions is owed.
// packet-audit:fname CUICashItemGachapon::OnButtonClicked
type CashItemGachaponButton struct {
	cashId int64
}

func NewCashItemGachaponButton(cashId int64) CashItemGachaponButton {
	return CashItemGachaponButton{cashId: cashId}
}

func (m CashItemGachaponButton) CashId() int64 { return m.cashId }

func (m CashItemGachaponButton) Operation() string { return CashItemGachaponHandle }

func (m CashItemGachaponButton) String() string {
	return fmt.Sprintf("cashId [%d]", m.cashId)
}

func (m CashItemGachaponButton) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt64(m.cashId)
		return w.Bytes()
	}
}

func (m *CashItemGachaponButton) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.cashId = r.ReadInt64()
	}
}
