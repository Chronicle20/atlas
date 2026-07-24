package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const ShopScannerItemUseHandle = "ShopScannerItemUseHandle"

// packet-audit:fname CWvsContext::SendShopScannerItemUseRequest
// ShopScannerItemUse is the dedicated use-route for the USE-inventory owl
// (231xxxx family), double-clicked from the inventory. Gated client-side on
// itemId/10000 == 231 (v95 is_shopscanner_item 0x4ff5c0). No leading
// updateTime on any version, v95 included (verified 0x9e10e0; v83 0xa0a25e).
//
// The pre-v83 GMS clients (verified v79 @0x9703a3, v72 @0x91e45b) send an
// earlier, shorter frame: [str serial][short pos][int itemId] with NO
// searchItemId/descending/updateTime — the item is used immediately from
// CDraggableItem::OnDoubleClicked and carries its cash-commodity serial string
// instead of the scanner-input search parameters (there is no CUIShopScanner
// input dialog before v83). The serial-bearing legacy frame therefore has no
// server-side search target; searchItemId stays 0 on that path. v61/v48 have no
// dedicated sender at all (not routed).
type ShopScannerItemUse struct {
	serial       string
	source       int16
	itemId       uint32
	searchItemId uint32
	descending   bool
	updateTime   uint32
}

func NewShopScannerItemUse(source int16, itemId uint32, searchItemId uint32, descending bool, updateTime uint32) ShopScannerItemUse {
	return ShopScannerItemUse{source: source, itemId: itemId, searchItemId: searchItemId, descending: descending, updateTime: updateTime}
}

// itemUseLegacyFrame reports whether the pre-v83 [str serial][short pos][int
// itemId] wire layout applies (verified v72/v79). GMS v83 onward and JMS use the
// [short source][int itemId][int searchItemId][bool descending][int updateTime]
// frame.
func itemUseLegacyFrame(t tenant.Model) bool {
	return t.Region() == "GMS" && t.MajorVersion() < 83
}

func (m ShopScannerItemUse) Serial() string {
	return m.serial
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
	return fmt.Sprintf("serial [%s] source [%d] itemId [%d] searchItemId [%d] descending [%t] updateTime [%d]", m.serial, m.source, m.itemId, m.searchItemId, m.descending, m.updateTime)
}

func (m ShopScannerItemUse) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if itemUseLegacyFrame(t) {
			w.WriteAsciiString(m.serial)
			w.WriteInt16(m.source)
			w.WriteInt(m.itemId)
			return w.Bytes()
		}
		w.WriteInt16(m.source)
		w.WriteInt(m.itemId)
		w.WriteInt(m.searchItemId)
		w.WriteBool(m.descending)
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *ShopScannerItemUse) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if itemUseLegacyFrame(t) {
			m.serial = r.ReadAsciiString()
			m.source = r.ReadInt16()
			m.itemId = r.ReadUint32()
			return
		}
		m.source = r.ReadInt16()
		m.itemId = r.ReadUint32()
		m.searchItemId = r.ReadUint32()
		m.descending = r.ReadBool()
		m.updateTime = r.ReadUint32()
	}
}
