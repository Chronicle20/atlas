package clientbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// scannerResultHasNpcShopPrice reports whether the RESULT-mode frame carries the
// leading nNpcShopPrice int. The pre-v72 GMS clients (verified v61
// CWvsContext::OnShopScannerResult @0x849800) read a 2-int header
// [int nItemID][int nCount] with no synthetic-NPC price field; v72 onward
// (verified v72 @0x920d9f, matches v83) read the 3-int
// [int nNpcShopPrice][int nItemID][int nCount] header.
func scannerResultHasNpcShopPrice(t tenant.Model) bool {
	return !(t.Region() == "GMS" && t.MajorVersion() < 72)
}

const ShopScannerResultWriter = "ShopScannerResult"

// ShopScannerRecord is one row of the shop-scanner result list
// (CWvsContext::OnShopScannerResult mode 6, ITEMDATA in the v95 typed struct).
// dwMiniRoomSN carries the shop-owner characterId (task-127
// design §4.4); channelId is 0-based on the wire; asset must be a
// zeroPosition (slotless) model.Asset and is encoded only when
// inventoryType == 1 (equip).
type ShopScannerRecord struct {
	ownerName     string
	mapId         uint32
	title         string
	bundles       uint32
	bundleSize    uint32
	price         uint32
	ownerId       uint32
	channelId     byte
	inventoryType byte
	asset         *model.Asset
}

func NewShopScannerRecord(ownerName string, mapId uint32, title string, bundles uint32, bundleSize uint32, price uint32, ownerId uint32, channelId byte, inventoryType byte, asset *model.Asset) ShopScannerRecord {
	return ShopScannerRecord{
		ownerName:     ownerName,
		mapId:         mapId,
		title:         title,
		bundles:       bundles,
		bundleSize:    bundleSize,
		price:         price,
		ownerId:       ownerId,
		channelId:     channelId,
		inventoryType: inventoryType,
		asset:         asset,
	}
}

func (r ShopScannerRecord) OwnerName() string   { return r.ownerName }
func (r ShopScannerRecord) MapId() uint32       { return r.mapId }
func (r ShopScannerRecord) Title() string       { return r.title }
func (r ShopScannerRecord) Bundles() uint32     { return r.bundles }
func (r ShopScannerRecord) BundleSize() uint32  { return r.bundleSize }
func (r ShopScannerRecord) Price() uint32       { return r.price }
func (r ShopScannerRecord) OwnerId() uint32     { return r.ownerId }
func (r ShopScannerRecord) ChannelId() byte     { return r.channelId }
func (r ShopScannerRecord) InventoryType() byte { return r.inventoryType }
func (r ShopScannerRecord) Asset() *model.Asset { return r.asset }

// packet-audit:fname CWvsContext::OnShopScannerResult#Result
// ShopScannerResult is mode 6 of CWvsContext::OnShopScannerResult (v83
// 0xa28c29, v95 0xa076c0). nNpcShopPrice > 0 makes the client insert a
// synthetic "sold in regular stores" first row — Atlas always sends 0.
// nCount==0 && nNpcShopPrice==0 shows the faithful no-results message.
type ShopScannerResult struct {
	mode         byte
	npcShopPrice uint32
	itemId       uint32
	records      []ShopScannerRecord
}

func NewShopScannerResult(mode byte, itemId uint32, records []ShopScannerRecord) ShopScannerResult {
	return ShopScannerResult{mode: mode, npcShopPrice: 0, itemId: itemId, records: records}
}

func (m ShopScannerResult) Mode() byte                   { return m.mode }
func (m ShopScannerResult) ItemId() uint32               { return m.itemId }
func (m ShopScannerResult) Records() []ShopScannerRecord { return m.records }

func (m ShopScannerResult) Operation() string {
	return ShopScannerResultWriter
}

func (m ShopScannerResult) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		if scannerResultHasNpcShopPrice(t) {
			w.WriteInt(m.npcShopPrice)
		}
		w.WriteInt(m.itemId)
		w.WriteInt(uint32(len(m.records)))
		for _, rec := range m.records {
			w.WriteAsciiString(rec.ownerName)
			w.WriteInt(rec.mapId)
			w.WriteAsciiString(rec.title)
			w.WriteInt(rec.bundles)
			w.WriteInt(rec.bundleSize)
			w.WriteInt(rec.price)
			w.WriteInt(rec.ownerId)
			w.WriteByte(rec.channelId)
			w.WriteByte(rec.inventoryType)
			if rec.inventoryType == 1 && rec.asset != nil {
				w.WriteByteArray(rec.asset.Encode(l, ctx)(options))
			}
		}
		return w.Bytes()
	}
}

func (m *ShopScannerResult) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		if scannerResultHasNpcShopPrice(t) {
			m.npcShopPrice = r.ReadUint32()
		}
		m.itemId = r.ReadUint32()
		count := r.ReadUint32()
		m.records = make([]ShopScannerRecord, 0, count)
		for i := uint32(0); i < count; i++ {
			var rec ShopScannerRecord
			rec.ownerName = r.ReadAsciiString()
			rec.mapId = r.ReadUint32()
			rec.title = r.ReadAsciiString()
			rec.bundles = r.ReadUint32()
			rec.bundleSize = r.ReadUint32()
			rec.price = r.ReadUint32()
			rec.ownerId = r.ReadUint32()
			rec.channelId = r.ReadByte()
			rec.inventoryType = r.ReadByte()
			if rec.inventoryType == 1 {
				a := &model.Asset{}
				a.Decode(l, ctx)(r, options)
				rec.asset = a
			}
			m.records = append(m.records, rec)
		}
	}
}

// packet-audit:fname CWvsContext::OnShopScannerResult#HotList
// ShopScannerHotList is mode 7: the most-searched item list shown when the
// scanner UI opens. Short lists (fewer than 10 ever-searched items) send the
// actual count — no filler.
type ShopScannerHotList struct {
	mode    byte
	itemIds []uint32
}

func NewShopScannerHotList(mode byte, itemIds []uint32) ShopScannerHotList {
	return ShopScannerHotList{mode: mode, itemIds: itemIds}
}

func (m ShopScannerHotList) Mode() byte        { return m.mode }
func (m ShopScannerHotList) ItemIds() []uint32 { return m.itemIds }

func (m ShopScannerHotList) Operation() string {
	return ShopScannerResultWriter
}

func (m ShopScannerHotList) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(byte(len(m.itemIds)))
		for _, id := range m.itemIds {
			w.WriteInt(id)
		}
		return w.Bytes()
	}
}

func (m *ShopScannerHotList) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := r.ReadByte()
		m.itemIds = make([]uint32, 0, count)
		for i := byte(0); i < count; i++ {
			m.itemIds = append(m.itemIds, r.ReadUint32())
		}
	}
}
