package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ShopScannerItemUseHandle = "ShopScannerItemUseHandle"

// packet-audit:fname CWvsContext::SendShopScannerItemUseRequest
// ShopScannerItemUse is the dedicated use-route for the USE-inventory owl
// (231xxxx family), double-clicked from the inventory. Gated client-side on
// itemId/10000 == 231 (v95 is_shopscanner_item 0x4ff5c0). No leading
// updateTime on any version, v95 included (verified 0x9e10e0; v83 0xa0a25e).
type ShopScannerItemUse struct {
	source       int16
	itemId       uint32
	searchItemId uint32
	descending   bool
	updateTime   uint32
}

func NewShopScannerItemUse(source int16, itemId uint32, searchItemId uint32, descending bool, updateTime uint32) ShopScannerItemUse {
	return ShopScannerItemUse{source: source, itemId: itemId, searchItemId: searchItemId, descending: descending, updateTime: updateTime}
}

func (m ShopScannerItemUse) Source() int16 {
	return m.source
}

func (m ShopScannerItemUse) ItemId() uint32 {
	return m.itemId
}

func (m ShopScannerItemUse) SearchItemId() uint32 {
	return m.searchItemId
}

func (m ShopScannerItemUse) Descending() bool {
	return m.descending
}

func (m ShopScannerItemUse) UpdateTime() uint32 {
	return m.updateTime
}

func (m ShopScannerItemUse) Operation() string {
	return ShopScannerItemUseHandle
}

func (m ShopScannerItemUse) String() string {
	return fmt.Sprintf("source [%d] itemId [%d] searchItemId [%d] descending [%t] updateTime [%d]", m.source, m.itemId, m.searchItemId, m.descending, m.updateTime)
}

func (m ShopScannerItemUse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.source)
		w.WriteInt(m.itemId)
		w.WriteInt(m.searchItemId)
		w.WriteBool(m.descending)
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *ShopScannerItemUse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.source = r.ReadInt16()
		m.itemId = r.ReadUint32()
		m.searchItemId = r.ReadUint32()
		m.descending = r.ReadBool()
		m.updateTime = r.ReadUint32()
	}
}
