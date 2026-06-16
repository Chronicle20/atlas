package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// List/item-blob body arms of the CITC::OnNormalItemResult dispatcher
// (MTS_OPERATION). These are the heavy arms that embed one or more ITCITEM
// entries — each ITCITEM wraps a GW_ItemSlotBase item blob (modeled by the
// shared model.Asset codec) plus an MTS trailer of meso/contract/bid metadata.
//
// All addresses are v95 (PDB-authoritative, port 13340). The read order is
// version-stable across gms_v83/v84/v87/v95 — the per-version sub-handler
// addresses are pinned in the byte-fixture verify markers
// (mts_operation_list_test.go). jms_v185 has no CITC op (registry-absent).
//
// ITCITEM::Decode (v95 0x575710) read order, per the decompile, AFTER the
// embedded GW_ItemSlotBase::Decode (v95 0x4f7ea0 -> Decode1 typeByte +
// RawDecode; modeled by model.Asset.Decode):
//
//	Decode4 nITCSN
//	Decode4 nPrice
//	Decode4 nContractFee
//	DecodeStr sContractFeeTxId
//	DecodeStr sRollbackUsageID
//	DecodeBuffer(8) ftITCDateExpired   (FILETIME)
//	DecodeStr sUserID
//	DecodeStr sGameID
//	DecodeStr sComment
//	Decode4 nBidCount
//	Decode4 nBidRange
//	Decode4 nBidPrice
//	Decode4 nMinPrice
//	Decode4 nMaxPrice
//	Decode4 nUnitPrice
//	Decode2 nProcessStatus

// MtsItem models one ITCITEM entry: the GW_ItemSlotBase item blob followed by
// the MTS trailer. The item blob is encoded/decoded by the shared model.Asset
// codec (the verified GW_ItemSlotBase contract); the trailer fields below are
// cited 1:1 to ITCITEM::Decode (v95 0x575710).
type MtsItem struct {
	item          model.Asset // GW_ItemSlotBase::Decode (Decode1 type + RawDecode)
	itcSn         uint32      // Decode4 nITCSN
	price         uint32      // Decode4 nPrice
	contractFee   uint32      // Decode4 nContractFee
	contractFeeTx string      // DecodeStr sContractFeeTxId
	rollbackUsage string      // DecodeStr sRollbackUsageID
	dateExpired   [8]byte     // DecodeBuffer(8) ftITCDateExpired (FILETIME)
	userId        string      // DecodeStr sUserID
	gameId        string      // DecodeStr sGameID
	comment       string      // DecodeStr sComment
	bidCount      uint32      // Decode4 nBidCount
	bidRange      uint32      // Decode4 nBidRange
	bidPrice      uint32      // Decode4 nBidPrice
	minPrice      uint32      // Decode4 nMinPrice
	maxPrice      uint32      // Decode4 nMaxPrice
	unitPrice     uint32      // Decode4 nUnitPrice
	processStatus uint16      // Decode2 nProcessStatus
}

func NewMtsItem(item model.Asset, itcSn uint32, price uint32, contractFee uint32, contractFeeTx string, rollbackUsage string, dateExpired [8]byte, userId string, gameId string, comment string, bidCount uint32, bidRange uint32, bidPrice uint32, minPrice uint32, maxPrice uint32, unitPrice uint32, processStatus uint16) MtsItem {
	return MtsItem{
		item:          item,
		itcSn:         itcSn,
		price:         price,
		contractFee:   contractFee,
		contractFeeTx: contractFeeTx,
		rollbackUsage: rollbackUsage,
		dateExpired:   dateExpired,
		userId:        userId,
		gameId:        gameId,
		comment:       comment,
		bidCount:      bidCount,
		bidRange:      bidRange,
		bidPrice:      bidPrice,
		minPrice:      minPrice,
		maxPrice:      maxPrice,
		unitPrice:     unitPrice,
		processStatus: processStatus,
	}
}

func (m MtsItem) Item() model.Asset     { return m.item }
func (m MtsItem) ItcSn() uint32         { return m.itcSn }
func (m MtsItem) Price() uint32         { return m.price }
func (m MtsItem) ContractFee() uint32   { return m.contractFee }
func (m MtsItem) ContractFeeTx() string { return m.contractFeeTx }
func (m MtsItem) RollbackUsage() string { return m.rollbackUsage }
func (m MtsItem) DateExpired() [8]byte  { return m.dateExpired }
func (m MtsItem) UserId() string        { return m.userId }
func (m MtsItem) GameId() string        { return m.gameId }
func (m MtsItem) Comment() string       { return m.comment }
func (m MtsItem) BidCount() uint32      { return m.bidCount }
func (m MtsItem) BidRange() uint32      { return m.bidRange }
func (m MtsItem) BidPrice() uint32      { return m.bidPrice }
func (m MtsItem) MinPrice() uint32      { return m.minPrice }
func (m MtsItem) MaxPrice() uint32      { return m.maxPrice }
func (m MtsItem) UnitPrice() uint32     { return m.unitPrice }
func (m MtsItem) ProcessStatus() uint16 { return m.processStatus }

// Encode writes one ITCITEM. The leading GW_ItemSlotBase blob is encoded by the
// shared model.Asset codec via WriteByteArray (the standard recurse pattern);
// each remaining line maps to a CInPacket::Decode* in ITCITEM::Decode
// (v95 0x575710).
func (m MtsItem) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		itemCopy := m.item
		w.WriteByteArray(itemCopy.Encode(l, ctx)(options)) // GW_ItemSlotBase::Decode (model.Asset codec)
		w.WriteInt(m.itcSn)                                // Decode4 nITCSN
		w.WriteInt(m.price)                                // Decode4 nPrice
		w.WriteInt(m.contractFee)                          // Decode4 nContractFee
		w.WriteAsciiString(m.contractFeeTx)                // DecodeStr sContractFeeTxId
		w.WriteAsciiString(m.rollbackUsage)                // DecodeStr sRollbackUsageID
		w.WriteByteArray(m.dateExpired[:])                 // DecodeBuffer(8) ftITCDateExpired
		w.WriteAsciiString(m.userId)                       // DecodeStr sUserID
		w.WriteAsciiString(m.gameId)                       // DecodeStr sGameID
		w.WriteAsciiString(m.comment)                      // DecodeStr sComment
		w.WriteInt(m.bidCount)                             // Decode4 nBidCount
		w.WriteInt(m.bidRange)                             // Decode4 nBidRange
		w.WriteInt(m.bidPrice)                             // Decode4 nBidPrice
		w.WriteInt(m.minPrice)                             // Decode4 nMinPrice
		w.WriteInt(m.maxPrice)                             // Decode4 nMaxPrice
		w.WriteInt(m.unitPrice)                            // Decode4 nUnitPrice
		w.WriteShort(m.processStatus)                      // Decode2 nProcessStatus
		return w.Bytes()
	}
}

// Decode reads one ITCITEM (mirror of Encode).
func (m *MtsItem) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.item.Decode(l, ctx)(r, options)
		m.itcSn = r.ReadUint32()
		m.price = r.ReadUint32()
		m.contractFee = r.ReadUint32()
		m.contractFeeTx = r.ReadAsciiString()
		m.rollbackUsage = r.ReadAsciiString()
		copy(m.dateExpired[:], r.ReadBytes(8))
		m.userId = r.ReadAsciiString()
		m.gameId = r.ReadAsciiString()
		m.comment = r.ReadAsciiString()
		m.bidCount = r.ReadUint32()
		m.bidRange = r.ReadUint32()
		m.bidPrice = r.ReadUint32()
		m.minPrice = r.ReadUint32()
		m.maxPrice = r.ReadUint32()
		m.unitPrice = r.ReadUint32()
		m.processStatus = r.ReadUint16()
	}
}

// MtsResultGetItcListDone — the 0x15 GET_ITC_LIST_DONE arm
// (CITC::OnGetITCListDone, v95 0x576500). The sub-handler read order is:
//
//	Decode4 categoryItemCnt   (m_nCurrentCategoryItemCnt)
//	Decode4 pageItemCnt       (m_nCurrentPageItemCnt; loop count)
//	Decode4 category          (m_nCurCategory)
//	Decode4 subCategory       (m_nCurCategorySub)
//	Decode4 page              (m_nCurPage)
//	Decode1 sortType          (m_wndSubTab.m_nSortType)
//	Decode1 sortColumn        (m_wndSubTab.m_nSortColumn)
//	pageItemCnt × ITCITEM::Decode
//	Decode1 requestSent       (if nonzero, m_bITCRequestSent = 0)
//
// The codec writes the mode byte (0x15) THEN this body. len(items) is the
// pageItemCnt loop count.
//
// packet-audit:fname CITC::OnNormalItemResult#GetItcListDone
type MtsResultGetItcListDone struct {
	mode            byte
	categoryItemCnt uint32
	category        uint32
	subCategory     uint32
	page            uint32
	sortType        byte
	sortColumn      byte
	items           []MtsItem
	requestSent     byte
}

func NewMtsResultGetItcListDone(categoryItemCnt uint32, category uint32, subCategory uint32, page uint32, sortType byte, sortColumn byte, items []MtsItem, requestSent byte) MtsResultGetItcListDone {
	return MtsResultGetItcListDone{
		mode:            0x15,
		categoryItemCnt: categoryItemCnt,
		category:        category,
		subCategory:     subCategory,
		page:            page,
		sortType:        sortType,
		sortColumn:      sortColumn,
		items:           items,
		requestSent:     requestSent,
	}
}

func (m MtsResultGetItcListDone) Mode() byte              { return m.mode }
func (m MtsResultGetItcListDone) CategoryItemCnt() uint32 { return m.categoryItemCnt }
func (m MtsResultGetItcListDone) Category() uint32        { return m.category }
func (m MtsResultGetItcListDone) SubCategory() uint32     { return m.subCategory }
func (m MtsResultGetItcListDone) Page() uint32            { return m.page }
func (m MtsResultGetItcListDone) SortType() byte          { return m.sortType }
func (m MtsResultGetItcListDone) SortColumn() byte        { return m.sortColumn }
func (m MtsResultGetItcListDone) Items() []MtsItem        { return m.items }
func (m MtsResultGetItcListDone) RequestSent() byte       { return m.requestSent }
func (m MtsResultGetItcListDone) Operation() string       { return MtsOperationWriter }
func (m MtsResultGetItcListDone) String() string {
	return fmt.Sprintf("mts get itc list done mode [%d] category [%d] page [%d] items [%d]", m.mode, m.category, m.page, len(m.items))
}

func (m MtsResultGetItcListDone) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)              // dispatcher mode byte (0x15)
		w.WriteInt(m.categoryItemCnt)    // Decode4 categoryItemCnt
		w.WriteInt(uint32(len(m.items))) // Decode4 pageItemCnt (loop count)
		w.WriteInt(m.category)           // Decode4 category
		w.WriteInt(m.subCategory)        // Decode4 subCategory
		w.WriteInt(m.page)               // Decode4 page
		w.WriteByte(m.sortType)          // Decode1 sortType
		w.WriteByte(m.sortColumn)        // Decode1 sortColumn
		for _, it := range m.items {
			w.WriteByteArray(it.Encode(l, ctx)(options)) // ITCITEM::Decode
		}
		w.WriteByte(m.requestSent) // Decode1 requestSent
		return w.Bytes()
	}
}

func (m *MtsResultGetItcListDone) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.categoryItemCnt = r.ReadUint32()
		count := r.ReadUint32()
		m.category = r.ReadUint32()
		m.subCategory = r.ReadUint32()
		m.page = r.ReadUint32()
		m.sortType = r.ReadByte()
		m.sortColumn = r.ReadByte()
		m.items = make([]MtsItem, 0, count)
		for i := uint32(0); i < count; i++ {
			var it MtsItem
			it.Decode(l, ctx)(r, options)
			m.items = append(m.items, it)
		}
		m.requestSent = r.ReadByte()
	}
}

// MtsResultGetSearchItcListDone — the 0x17 GET_SEARCH_ITC_LIST_DONE arm
// (CITC::OnGetSearchITCListDone, v95 0x5766e0). The sub-handler read order is:
//
//	Decode4 categoryItemCnt   (m_nCurrentCategoryItemCnt)
//	Decode4 pageItemCnt       (m_nCurrentPageItemCnt; loop count)
//	Decode4 category          (m_nCurCategory)
//	Decode4 subCategory       (m_nCurCategorySub)
//	Decode4 page              (m_nCurPage)
//	pageItemCnt × ITCITEM::Decode
//
// Unlike GET_ITC_LIST_DONE it reads NO sortType/sortColumn bytes and NO
// trailing requestSent byte (it sets m_bITCRequestSent=0 unconditionally as a
// member store, not a wire read). The codec writes the mode byte (0x17) THEN
// this body.
//
// packet-audit:fname CITC::OnNormalItemResult#GetSearchItcListDone
type MtsResultGetSearchItcListDone struct {
	mode            byte
	categoryItemCnt uint32
	category        uint32
	subCategory     uint32
	page            uint32
	items           []MtsItem
}

func NewMtsResultGetSearchItcListDone(categoryItemCnt uint32, category uint32, subCategory uint32, page uint32, items []MtsItem) MtsResultGetSearchItcListDone {
	return MtsResultGetSearchItcListDone{
		mode:            0x17,
		categoryItemCnt: categoryItemCnt,
		category:        category,
		subCategory:     subCategory,
		page:            page,
		items:           items,
	}
}

func (m MtsResultGetSearchItcListDone) Mode() byte              { return m.mode }
func (m MtsResultGetSearchItcListDone) CategoryItemCnt() uint32 { return m.categoryItemCnt }
func (m MtsResultGetSearchItcListDone) Category() uint32        { return m.category }
func (m MtsResultGetSearchItcListDone) SubCategory() uint32     { return m.subCategory }
func (m MtsResultGetSearchItcListDone) Page() uint32            { return m.page }
func (m MtsResultGetSearchItcListDone) Items() []MtsItem        { return m.items }
func (m MtsResultGetSearchItcListDone) Operation() string       { return MtsOperationWriter }
func (m MtsResultGetSearchItcListDone) String() string {
	return fmt.Sprintf("mts get search itc list done mode [%d] category [%d] page [%d] items [%d]", m.mode, m.category, m.page, len(m.items))
}

func (m MtsResultGetSearchItcListDone) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)              // dispatcher mode byte (0x17)
		w.WriteInt(m.categoryItemCnt)    // Decode4 categoryItemCnt
		w.WriteInt(uint32(len(m.items))) // Decode4 pageItemCnt (loop count)
		w.WriteInt(m.category)           // Decode4 category
		w.WriteInt(m.subCategory)        // Decode4 subCategory
		w.WriteInt(m.page)               // Decode4 page
		for _, it := range m.items {
			w.WriteByteArray(it.Encode(l, ctx)(options)) // ITCITEM::Decode
		}
		return w.Bytes()
	}
}

func (m *MtsResultGetSearchItcListDone) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.categoryItemCnt = r.ReadUint32()
		count := r.ReadUint32()
		m.category = r.ReadUint32()
		m.subCategory = r.ReadUint32()
		m.page = r.ReadUint32()
		m.items = make([]MtsItem, 0, count)
		for i := uint32(0); i < count; i++ {
			var it MtsItem
			it.Decode(l, ctx)(r, options)
			m.items = append(m.items, it)
		}
	}
}

// MtsResultGetUserPurchaseItemDone — the 0x21 GET_USER_PURCHASE_ITEM_DONE arm
// (CITC::OnGetUserPurchaseItemDone, v95 0x576cf0). The sub-handler read order:
//
//	Decode4 totalCount        (loop count; m_aPurchaseItem)
//	totalCount × ITCITEM::Decode
//	Decode4 limitedCount      (read unconditionally; gates a StringPool notice)
//	Decode1 requestSent       (if nonzero, m_bITCRequestSent = 0)
//
// The codec writes the mode byte (0x21) THEN this body. len(items) is
// totalCount.
//
// packet-audit:fname CITC::OnNormalItemResult#GetUserPurchaseItemDone
type MtsResultGetUserPurchaseItemDone struct {
	mode         byte
	items        []MtsItem
	limitedCount uint32
	requestSent  byte
}

func NewMtsResultGetUserPurchaseItemDone(items []MtsItem, limitedCount uint32, requestSent byte) MtsResultGetUserPurchaseItemDone {
	return MtsResultGetUserPurchaseItemDone{
		mode:         0x21,
		items:        items,
		limitedCount: limitedCount,
		requestSent:  requestSent,
	}
}

func (m MtsResultGetUserPurchaseItemDone) Mode() byte           { return m.mode }
func (m MtsResultGetUserPurchaseItemDone) Items() []MtsItem     { return m.items }
func (m MtsResultGetUserPurchaseItemDone) LimitedCount() uint32 { return m.limitedCount }
func (m MtsResultGetUserPurchaseItemDone) RequestSent() byte    { return m.requestSent }
func (m MtsResultGetUserPurchaseItemDone) Operation() string    { return MtsOperationWriter }
func (m MtsResultGetUserPurchaseItemDone) String() string {
	return fmt.Sprintf("mts get user purchase item done mode [%d] items [%d] limited [%d]", m.mode, len(m.items), m.limitedCount)
}

func (m MtsResultGetUserPurchaseItemDone) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)              // dispatcher mode byte (0x21)
		w.WriteInt(uint32(len(m.items))) // Decode4 totalCount (loop count)
		for _, it := range m.items {
			w.WriteByteArray(it.Encode(l, ctx)(options)) // ITCITEM::Decode
		}
		w.WriteInt(m.limitedCount) // Decode4 limitedCount
		w.WriteByte(m.requestSent) // Decode1 requestSent
		return w.Bytes()
	}
}

func (m *MtsResultGetUserPurchaseItemDone) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := r.ReadUint32()
		m.items = make([]MtsItem, 0, count)
		for i := uint32(0); i < count; i++ {
			var it MtsItem
			it.Decode(l, ctx)(r, options)
			m.items = append(m.items, it)
		}
		m.limitedCount = r.ReadUint32()
		m.requestSent = r.ReadByte()
	}
}

// MtsResultGetUserSaleItemDone — the 0x23 GET_USER_SALE_ITEM_DONE arm
// (CITC::OnGetUserSaleItemDone, v95 0x576870). The sub-handler read order:
//
//	Decode4 totalCount        (loop count; m_aSaleItem)
//	totalCount × ITCITEM::Decode
//
// No trailing fields (the only further work is m_wndSale.Draw, no wire read).
// The codec writes the mode byte (0x23) THEN this body. len(items) is
// totalCount.
//
// packet-audit:fname CITC::OnNormalItemResult#GetUserSaleItemDone
type MtsResultGetUserSaleItemDone struct {
	mode  byte
	items []MtsItem
}

func NewMtsResultGetUserSaleItemDone(items []MtsItem) MtsResultGetUserSaleItemDone {
	return MtsResultGetUserSaleItemDone{mode: 0x23, items: items}
}

func (m MtsResultGetUserSaleItemDone) Mode() byte        { return m.mode }
func (m MtsResultGetUserSaleItemDone) Items() []MtsItem  { return m.items }
func (m MtsResultGetUserSaleItemDone) Operation() string { return MtsOperationWriter }
func (m MtsResultGetUserSaleItemDone) String() string {
	return fmt.Sprintf("mts get user sale item done mode [%d] items [%d]", m.mode, len(m.items))
}

func (m MtsResultGetUserSaleItemDone) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)              // dispatcher mode byte (0x23)
		w.WriteInt(uint32(len(m.items))) // Decode4 totalCount (loop count)
		for _, it := range m.items {
			w.WriteByteArray(it.Encode(l, ctx)(options)) // ITCITEM::Decode
		}
		return w.Bytes()
	}
}

func (m *MtsResultGetUserSaleItemDone) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := r.ReadUint32()
		m.items = make([]MtsItem, 0, count)
		for i := uint32(0); i < count; i++ {
			var it MtsItem
			it.Decode(l, ctx)(r, options)
			m.items = append(m.items, it)
		}
	}
}

// MtsResultLoadWishSaleListDone — the 0x2D LOAD_WISH_SALE_LIST_DONE arm
// (CITC::OnLoadWishSaleListDone, v95 0x5769a0). The sub-handler read order:
//
//	Decode4 totalCount        (loop count; m_aWishItem)
//	totalCount × ITCITEM::Decode
//
// No trailing fields (the count==0 / count>0 branches drive a notice or a modal
// dialog respectively, neither of which reads further bytes; m_bITCRequestSent
// is a member store). The codec writes the mode byte (0x2D) THEN this body.
// len(items) is totalCount.
//
// packet-audit:fname CITC::OnNormalItemResult#LoadWishSaleListDone
type MtsResultLoadWishSaleListDone struct {
	mode  byte
	items []MtsItem
}

func NewMtsResultLoadWishSaleListDone(items []MtsItem) MtsResultLoadWishSaleListDone {
	return MtsResultLoadWishSaleListDone{mode: 0x2D, items: items}
}

func (m MtsResultLoadWishSaleListDone) Mode() byte        { return m.mode }
func (m MtsResultLoadWishSaleListDone) Items() []MtsItem  { return m.items }
func (m MtsResultLoadWishSaleListDone) Operation() string { return MtsOperationWriter }
func (m MtsResultLoadWishSaleListDone) String() string {
	return fmt.Sprintf("mts load wish sale list done mode [%d] items [%d]", m.mode, len(m.items))
}

func (m MtsResultLoadWishSaleListDone) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)              // dispatcher mode byte (0x2D)
		w.WriteInt(uint32(len(m.items))) // Decode4 totalCount (loop count)
		for _, it := range m.items {
			w.WriteByteArray(it.Encode(l, ctx)(options)) // ITCITEM::Decode
		}
		return w.Bytes()
	}
}

func (m *MtsResultLoadWishSaleListDone) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := r.ReadUint32()
		m.items = make([]MtsItem, 0, count)
		for i := uint32(0); i < count; i++ {
			var it MtsItem
			it.Decode(l, ctx)(r, options)
			m.items = append(m.items, it)
		}
	}
}
