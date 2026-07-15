package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ItemUpgradeUpdateHandle = "ItemUpgradeUpdateHandle"

// ItemUpgradeUpdate — the CUIItemUpgrade gauge-confirm packet, sent once the
// dialog's gauge fills after the server armed it with the VICIOUS_HAMMER
// open-arm response. Reads (IDA v83 CUIItemUpgrade::Update 0x82ae28 /
// v95 0x7bef50): Encode4(m_nReturnResult) — the open-arm mode byte widened to
// int32 — then Encode4(m_nResult) — the server-chosen round-trip token, which
// packs hammerSlot(high int16) | equipSlot(low int16). Version-invariant.
// packet-audit:fname CUIItemUpgrade::Update
type ItemUpgradeUpdate struct {
	returnResult uint32
	result       uint32
}

func (m ItemUpgradeUpdate) ReturnResult() uint32 { return m.returnResult }
func (m ItemUpgradeUpdate) Result() uint32       { return m.result }

func (m ItemUpgradeUpdate) Operation() string { return ItemUpgradeUpdateHandle }

func (m ItemUpgradeUpdate) String() string {
	return fmt.Sprintf("returnResult [%d] result [%d]", m.returnResult, m.result)
}

func (m ItemUpgradeUpdate) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.returnResult)
		w.WriteInt(m.result)
		return w.Bytes()
	}
}

func (m *ItemUpgradeUpdate) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.returnResult = r.ReadUint32()
		m.result = r.ReadUint32()
	}
}
