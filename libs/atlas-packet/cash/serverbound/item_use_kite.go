package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ItemUseKite is the CashSlotItemType(18) sub-body of USE_CASH_ITEM — the
// message a player pins to a map with a category-508 kite (message box).
//
// Derived from the case-18 arm of CWvsContext::SendConsumeCashItemUseRequest
// (gms_v95 @0x9eb3e0; arm entry @0x9ecfa2). The arm builds a CUIHope dialog,
// reads its three edit controls via CUIHope::GetText @0x9ed0f8, joins them
// with '\n', screens the result through CCurseProcess::ProcessString @0x9ed1b7,
// and then performs its ONLY encode: COutPacket::EncodeStr @0x9ed271. So the
// sub-body is exactly one length-prefixed string.
//
// Placement coordinates are NOT on the wire — the server takes them from the
// character's own position. Nor is a kite type: the banner's appearance comes
// from the item id (see FieldKiteSpawn).
//
// updateTimeFirst mirrors ItemUse.UpdateTimeFirst: GMS <= v84 trails
// update_time after the sub-body, GMS v87+ and JMS lead it in the header.
type ItemUseKite struct {
	message         string
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseKite(updateTimeFirst bool) *ItemUseKite {
	return &ItemUseKite{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseKite) Message() string    { return m.message }
func (m ItemUseKite) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseKite) Operation() string { return "ItemUseKite" }

func (m ItemUseKite) String() string {
	return fmt.Sprintf("message [%s] updateTime [%d]", m.message, m.updateTime)
}

func (m ItemUseKite) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.message)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseKite) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.message = r.ReadAsciiString()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
