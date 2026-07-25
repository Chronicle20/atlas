package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// Discrete per-mode body codecs for the CITC::OnNormalItemResult dispatcher
// (MTS_OPERATION). The dispatcher reads Decode1(mode) and switch-dispatches to
// one of 35 sub-handlers (0x15..0x3E). All 35 codecs live in this one file
// (task-096 consolidation): the notice-only "Empty-shape" arms, the
// "Reason-shape" arms (Decode1 fail reason), the "TwoInts-shape" arms, the
// conditional-tail scalar arms (RegisterSaleEntryFailed / SuccessBidInfo), and
// the heavy list/item-blob arms (which embed the ITCITEM / GW_ItemSlotBase item
// blob via MtsItem). mts_operation2.go (MTS_OPERATION2, CITC::OnQueryCashResult)
// is a SEPARATE op and is not part of this file.
//
// Each struct's constructor takes the dispatcher mode byte as its FIRST argument
// (never hard-coded): the per-mode body functions in field/mts_operation_body.go
// resolve that byte from the tenant "operations" table (the same config-driven
// contract as the other dispatcher families — npc/storage/cash). Encode writes
// the leading mode byte THEN the full arm body (the read order the matching CITC
// sub-handler performs on CInPacket). The mode bytes are version-stable across
// gms_v83 (dispatcher 0x5a4311), gms_v84 (0x5b47c8), gms_v87 (0x5d43d0),
// gms_v95 (0x5771d0) — IDA-confirmed: the case labels (21..62) are identical and
// the sub-handler bodies are byte-identical in shape. jms_v185 has no CITC op
// (registry-absent).

// MtsOperationWriter is the registry writer name (Operation()) shared by every
// per-mode CITC::OnNormalItemResult (MTS_OPERATION) body codec in this file. The
// mode-only MtsOperation struct that originally declared this const was retired
// in task-096 once the per-mode body codecs made it dead; the const survives
// here because the codecs use it as their Operation().
const MtsOperationWriter = "MtsOperation"

// mtsRegisterSaleEntryFailedSaleLimitReason is the only reason value of the
// 0x1E RegisterSaleEntryFailed arm whose sub-handler reads a trailing Decode2
// short (the per-account sale-limit count). All other reasons route a plain
// NoticeFailReason with no trailing read.
//
//	v83 sub_5A4581:  if ( v3 == 72 ) { v4 = CInPacket::Decode2(a2); ... }
//	v84 sub_5B4A38:  if ( v3 == 72 ) { v4 = CInPacket::Decode2(a2); ... }
//	v87 OnNormalItemResRegisterSaleEntryFailed@0x5d4640: if (v3==72) Decode2
//	v95 OnNormalItemResRegisterSaleEntryFailed@0x576b80: if (v3==72) Decode2
const mtsRegisterSaleEntryFailedSaleLimitReason byte = 0x48 // 72

// MtsResultRegisterSaleEntryFailed — the 0x1E RegisterSaleEntryFailed arm
// (CITC::OnNormalItemResRegisterSaleEntryFailed). The sub-handler reads
// Decode1(reason) -> NoticeFailReason, EXCEPT when reason==0x48 (sale-limit
// reached) where it then reads a single Decode2 short (the limit count, used to
// Format the StringPool 0x12BB notice). The codec writes the mode byte, the
// Decode1 reason byte, and — ONLY when reason==0x48 — the Decode2 limit short.
//
// VERIFIED iteration 5 (task-096), decompiled in ALL FOUR versions (identical
// gate + read order; the trailing m_bITCRequestSent=0 store is not a wire read):
//
//	v83 sub_5A4581          / v84 sub_5B4A38
//	v87 0x5d4640            / v95 0x576b80
//
// packet-audit:fname CITC::OnNormalItemResult#RegisterSaleEntryFailed
type MtsResultRegisterSaleEntryFailed struct {
	mode      byte
	reason    byte
	saleLimit uint16 // Decode2; present on the wire only when reason==0x48
}

func NewMtsResultRegisterSaleEntryFailed(mode byte, reason byte, saleLimit uint16) MtsResultRegisterSaleEntryFailed {
	return MtsResultRegisterSaleEntryFailed{mode: mode, reason: reason, saleLimit: saleLimit}
}

func (m MtsResultRegisterSaleEntryFailed) Mode() byte        { return m.mode }
func (m MtsResultRegisterSaleEntryFailed) Reason() byte      { return m.reason }
func (m MtsResultRegisterSaleEntryFailed) SaleLimit() uint16 { return m.saleLimit }
func (m MtsResultRegisterSaleEntryFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultRegisterSaleEntryFailed) String() string {
	return fmt.Sprintf("mts register sale entry failed mode [%d] reason [%d] saleLimit [%d]", m.mode, m.reason, m.saleLimit)
}

func (m MtsResultRegisterSaleEntryFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte (0x1E)
		w.WriteByte(m.reason) // Decode1 fail reason -> NoticeFailReason
		if m.reason == mtsRegisterSaleEntryFailedSaleLimitReason {
			w.WriteShort(m.saleLimit) // Decode2 sale-limit count (reason==0x48 branch only)
		}
		return w.Bytes()
	}
}

func (m *MtsResultRegisterSaleEntryFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
		if m.reason == mtsRegisterSaleEntryFailedSaleLimitReason {
			m.saleLimit = r.ReadUint16()
		}
	}
}

// MtsResultSuccessBidInfo — the 0x3E SuccessBidInfoResult arm
// (CITC::OnSuccessBidInfoResult). The sub-handler reads Decode1(soldFlag) +
// Decode4(itemId); when itemId>0 it then reads Decode4(price) and an 8-byte
// FILETIME contract date via DecodeBuffer(8). When itemId<=0 the body ends after
// the leading byte+int (no notice shown). The codec writes the mode byte, the
// Decode1 soldFlag, the Decode4 itemId, and — ONLY when itemId>0 — the Decode4
// price followed by the 8-byte contract-date buffer.
//
// VERIFIED iteration 5 (task-096), decompiled in ALL FOUR versions (identical
// read order: Decode1, Decode4, [itemId>0: Decode4 + DecodeBuffer(8)]):
//
//	v83 sub_5A52DE          / v84 sub_5B5795
//	v87 OnSuccessBidInfoResult@0x5d5398 / v95 OnSuccessBidInfoResult@0x577000
//
// packet-audit:fname CITC::OnNormalItemResult#SuccessBidInfo
type MtsResultSuccessBidInfo struct {
	mode         byte
	soldFlag     byte    // Decode1: 1=sold (StringPool 0x12AA), else bought (0x12AB)
	itemId       uint32  // Decode4: ITC item id; <=0 ends the body
	price        uint32  // Decode4: meso price (itemId>0 only)
	contractDate [8]byte // DecodeBuffer(8): FILETIME (itemId>0 only)
}

func NewMtsResultSuccessBidInfo(mode byte, soldFlag byte, itemId uint32, price uint32, contractDate [8]byte) MtsResultSuccessBidInfo {
	return MtsResultSuccessBidInfo{mode: mode, soldFlag: soldFlag, itemId: itemId, price: price, contractDate: contractDate}
}

func (m MtsResultSuccessBidInfo) Mode() byte            { return m.mode }
func (m MtsResultSuccessBidInfo) SoldFlag() byte        { return m.soldFlag }
func (m MtsResultSuccessBidInfo) ItemId() uint32        { return m.itemId }
func (m MtsResultSuccessBidInfo) Price() uint32         { return m.price }
func (m MtsResultSuccessBidInfo) ContractDate() [8]byte { return m.contractDate }
func (m MtsResultSuccessBidInfo) Operation() string     { return MtsOperationWriter }
func (m MtsResultSuccessBidInfo) String() string {
	return fmt.Sprintf("mts success bid info mode [%d] soldFlag [%d] itemId [%d] price [%d]", m.mode, m.soldFlag, m.itemId, m.price)
}

func (m MtsResultSuccessBidInfo) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)     // dispatcher mode byte (0x3E)
		w.WriteByte(m.soldFlag) // Decode1 sold/bought flag
		w.WriteInt(m.itemId)    // Decode4 ITC item id
		if m.itemId > 0 {
			w.WriteInt(m.price)                 // Decode4 meso price (itemId>0 branch)
			w.WriteByteArray(m.contractDate[:]) // DecodeBuffer(8) FILETIME contract date
		}
		return w.Bytes()
	}
}

func (m *MtsResultSuccessBidInfo) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.soldFlag = r.ReadByte()
		m.itemId = r.ReadUint32()
		if m.itemId > 0 {
			m.price = r.ReadUint32()
			copy(m.contractDate[:], r.ReadBytes(8))
		}
	}
}

// List/item-blob body arms of the CITC::OnNormalItemResult dispatcher
// (MTS_OPERATION). These are the heavy arms that embed one or more ITCITEM
// entries — each ITCITEM wraps a GW_ItemSlotBase item blob (modeled by the
// shared model.Asset codec) plus an MTS trailer of meso/contract/bid metadata.
//
// All addresses are v95 (PDB-authoritative, port 13340). The read order is
// version-stable across gms_v83/v84/v87/v95 — the per-version sub-handler
// addresses are pinned in the byte-fixture verify markers
// (mts_operation_test.go). jms_v185 has no CITC op (registry-absent).
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
	processStatus uint16      // Decode2 nProcessStatus (raw wire value; the Decode target)
	// processStatusKey is the SEMANTIC history/auction-status key that Encode resolves
	// to the nProcessStatus wire code from the tenant `processStatusCodes` writer table
	// (DOM-25), so the client-switch code (CITCWnd_List::GetContractHistoryCode /
	// GetAuctionHistoryCode) is config-driven, not a Go literal. Empty key => code 0.
	processStatusKey string
}

// MtsProcessStatus* are the SEMANTIC keys for the nProcessStatus column, resolved
// per-tenant/version through the `processStatusCodes` writer-options table. The
// History tab feeds the code to GetContractHistoryCode (0=Sold/1=Purchased/2=Bid
// Lost/3=Cancelled) and the Auction tab to GetAuctionHistoryCode (1=Exhibit/2=Bid).
const (
	MtsProcessStatusNone             = ""                  // holdings/wishes: no status column => resolves to 0
	MtsProcessStatusHistorySold      = "HISTORY_SOLD"      // GetContractHistoryCode 0
	MtsProcessStatusHistoryPurchased = "HISTORY_PURCHASED" // GetContractHistoryCode 1
	MtsProcessStatusHistoryBidLost   = "HISTORY_BID_LOST"  // GetContractHistoryCode 2
	MtsProcessStatusHistoryCancelled = "HISTORY_CANCELLED" // GetContractHistoryCode 3
	MtsProcessStatusAuctionExhibit   = "AUCTION_EXHIBIT"   // GetAuctionHistoryCode 1
	MtsProcessStatusAuctionBid       = "AUCTION_BID"       // GetAuctionHistoryCode 2
)

// mtsProcessStatusDefaults are the IDA-verified, version-stable nProcessStatus wire
// codes (identical across gms v83/84/87/95). They live inside the codec (which
// DOM-25 permits for wire codes) as the fallback for a tenant whose
// processStatusCodes writer table is absent or incomplete — so the History
// disposition and Auction category columns render correctly BEFORE the config table
// is patched, instead of collapsing every row to 0 ("Sold"). A tenant that DOES
// provide the table overrides these (DOM-25: config is authoritative when present).
var mtsProcessStatusDefaults = map[string]uint16{
	MtsProcessStatusHistorySold:      0,
	MtsProcessStatusHistoryPurchased: 1,
	MtsProcessStatusHistoryBidLost:   2,
	MtsProcessStatusHistoryCancelled: 3,
	MtsProcessStatusAuctionExhibit:   1,
	MtsProcessStatusAuctionBid:       2,
}

// resolveProcessStatusCode resolves options["processStatusCodes"][key] to the
// nProcessStatus wire code, falling back to the built-in version-stable default when
// the tenant table is absent or lacks the key (an empty key => 0). Never crashes.
func resolveProcessStatusCode(options map[string]interface{}, key string) uint16 {
	if key == "" {
		return 0
	}
	if raw, ok := options["processStatusCodes"]; ok {
		if table, ok := raw.(map[string]interface{}); ok {
			switch n := table[key].(type) {
			case float64:
				return uint16(n)
			case int:
				return uint16(n)
			}
		}
	}
	return mtsProcessStatusDefaults[key]
}

func NewMtsItem(item model.Asset, itcSn uint32, price uint32, contractFee uint32, contractFeeTx string, rollbackUsage string, dateExpired [8]byte, userId string, gameId string, comment string, bidCount uint32, bidRange uint32, bidPrice uint32, minPrice uint32, maxPrice uint32, unitPrice uint32, processStatusKey string) MtsItem {
	return MtsItem{
		item:             item,
		itcSn:            itcSn,
		price:            price,
		contractFee:      contractFee,
		contractFeeTx:    contractFeeTx,
		rollbackUsage:    rollbackUsage,
		dateExpired:      dateExpired,
		userId:           userId,
		gameId:           gameId,
		comment:          comment,
		bidCount:         bidCount,
		bidRange:         bidRange,
		bidPrice:         bidPrice,
		minPrice:         minPrice,
		maxPrice:         maxPrice,
		unitPrice:        unitPrice,
		processStatusKey: processStatusKey,
	}
}

func (m MtsItem) Item() model.Asset        { return m.item }
func (m MtsItem) ItcSn() uint32            { return m.itcSn }
func (m MtsItem) Price() uint32            { return m.price }
func (m MtsItem) ContractFee() uint32      { return m.contractFee }
func (m MtsItem) ContractFeeTx() string    { return m.contractFeeTx }
func (m MtsItem) RollbackUsage() string    { return m.rollbackUsage }
func (m MtsItem) DateExpired() [8]byte     { return m.dateExpired }
func (m MtsItem) UserId() string           { return m.userId }
func (m MtsItem) GameId() string           { return m.gameId }
func (m MtsItem) Comment() string          { return m.comment }
func (m MtsItem) BidCount() uint32         { return m.bidCount }
func (m MtsItem) BidRange() uint32         { return m.bidRange }
func (m MtsItem) BidPrice() uint32         { return m.bidPrice }
func (m MtsItem) MinPrice() uint32         { return m.minPrice }
func (m MtsItem) MaxPrice() uint32         { return m.maxPrice }
func (m MtsItem) UnitPrice() uint32        { return m.unitPrice }
func (m MtsItem) ProcessStatus() uint16    { return m.processStatus }
func (m MtsItem) ProcessStatusKey() string { return m.processStatusKey }

// Encode writes one ITCITEM. The leading GW_ItemSlotBase blob is encoded by the
// shared model.Asset codec via WriteByteArray (the standard recurse pattern);
// each remaining line maps to a CInPacket::Decode* in ITCITEM::Decode
// (v95 0x575710).
func (m MtsItem) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		itemCopy := m.item
		w.WriteByteArray(itemCopy.Encode(l, ctx)(options))                  // GW_ItemSlotBase::Decode (model.Asset codec)
		w.WriteInt(m.itcSn)                                                 // Decode4 nITCSN
		w.WriteInt(m.price)                                                 // Decode4 nPrice
		w.WriteInt(m.contractFee)                                           // Decode4 nContractFee
		w.WriteAsciiString(m.contractFeeTx)                                 // DecodeStr sContractFeeTxId
		w.WriteAsciiString(m.rollbackUsage)                                 // DecodeStr sRollbackUsageID
		w.WriteByteArray(m.dateExpired[:])                                  // DecodeBuffer(8) ftITCDateExpired
		w.WriteAsciiString(m.userId)                                        // DecodeStr sUserID
		w.WriteAsciiString(m.gameId)                                        // DecodeStr sGameID
		w.WriteAsciiString(m.comment)                                       // DecodeStr sComment
		w.WriteInt(m.bidCount)                                              // Decode4 nBidCount
		w.WriteInt(m.bidRange)                                              // Decode4 nBidRange
		w.WriteInt(m.bidPrice)                                              // Decode4 nBidPrice
		w.WriteInt(m.minPrice)                                              // Decode4 nMinPrice
		w.WriteInt(m.maxPrice)                                              // Decode4 nMaxPrice
		w.WriteInt(m.unitPrice)                                             // Decode4 nUnitPrice
		w.WriteShort(resolveProcessStatusCode(options, m.processStatusKey)) // Decode2 nProcessStatus (config-resolved)
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

func NewMtsResultGetItcListDone(mode byte, categoryItemCnt uint32, category uint32, subCategory uint32, page uint32, sortType byte, sortColumn byte, items []MtsItem, requestSent byte) MtsResultGetItcListDone {
	return MtsResultGetItcListDone{
		mode:            mode,
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

func NewMtsResultGetSearchItcListDone(mode byte, categoryItemCnt uint32, category uint32, subCategory uint32, page uint32, items []MtsItem) MtsResultGetSearchItcListDone {
	return MtsResultGetSearchItcListDone{
		mode:            mode,
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

func NewMtsResultGetUserPurchaseItemDone(mode byte, items []MtsItem, limitedCount uint32, requestSent byte) MtsResultGetUserPurchaseItemDone {
	return MtsResultGetUserPurchaseItemDone{
		mode:         mode,
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

func NewMtsResultGetUserSaleItemDone(mode byte, items []MtsItem) MtsResultGetUserSaleItemDone {
	return MtsResultGetUserSaleItemDone{mode: mode, items: items}
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

func NewMtsResultLoadWishSaleListDone(mode byte, items []MtsItem) MtsResultLoadWishSaleListDone {
	return MtsResultLoadWishSaleListDone{mode: mode, items: items}
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

// Discrete per-mode notice-only ("Empty-shape") arms of the
// CITC::OnNormalItemResult dispatcher (MTS_OPERATION). Each arm's CITC
// sub-handler reads NOTHING from the packet after the dispatcher's Decode1(mode)
// — it shows a StringPool::GetString + CUtilDlg::Notice and clears
// m_bITCRequestSent (a member store, not a wire read). Each mode therefore has
// its OWN discrete struct that writes exactly that one mode byte (the byte its
// constructor receives from the config-resolved per-mode body function) — no shared shape struct (task-096: discrete-per-mode rule).
//
// The mode bytes are version-stable across gms_v83 / gms_v84 / gms_v87 /
// gms_v95 (IDA-verified; the case labels 0x15..0x3E are identical and the
// sub-handler bodies are byte-identical in shape). jms_v185 has NO CITC op
// (registry-absent). Per-mode per-version sub-handler addresses are cited in the
// doc comment above each struct (dispatcher: v83 0x5a4311 / v84 0x5b47c8 /
// v87 0x5d43d0 / v95 0x5771d0).

// MtsResultRegisterSaleEntryDone — the 0x1D RegisterSaleEntryDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a4674 / v84 0x5b4b64 / v87 0x5d4748 / v95 0x575cd0.
//
// packet-audit:fname CITC::OnNormalItemResult#RegisterSaleEntryDone
type MtsResultRegisterSaleEntryDone struct {
	mode byte
}

func NewMtsResultRegisterSaleEntryDone(mode byte) MtsResultRegisterSaleEntryDone {
	return MtsResultRegisterSaleEntryDone{mode: mode}
}

func (m MtsResultRegisterSaleEntryDone) Mode() byte        { return m.mode }
func (m MtsResultRegisterSaleEntryDone) Operation() string { return MtsOperationWriter }
func (m MtsResultRegisterSaleEntryDone) String() string {
	return fmt.Sprintf("mts register sale entry done mode [%d]", m.mode)
}

func (m MtsResultRegisterSaleEntryDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x1D); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultRegisterSaleEntryDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultSaleCurrentItemToWishDone — the 0x1F SaleCurrentItemToWishDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a46b2 / v84 0x5b4ba2 / v87 0x5d4786 / v95 0x575d20.
//
// packet-audit:fname CITC::OnNormalItemResult#SaleCurrentItemToWishDone
type MtsResultSaleCurrentItemToWishDone struct {
	mode byte
}

func NewMtsResultSaleCurrentItemToWishDone(mode byte) MtsResultSaleCurrentItemToWishDone {
	return MtsResultSaleCurrentItemToWishDone{mode: mode}
}

func (m MtsResultSaleCurrentItemToWishDone) Mode() byte        { return m.mode }
func (m MtsResultSaleCurrentItemToWishDone) Operation() string { return MtsOperationWriter }
func (m MtsResultSaleCurrentItemToWishDone) String() string {
	return fmt.Sprintf("mts sale current item to wish done mode [%d]", m.mode)
}

func (m MtsResultSaleCurrentItemToWishDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x1F); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultSaleCurrentItemToWishDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultCancelSaleItemDone — the 0x25 CancelSaleItemDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a4d14 / v84 0x5b5204 / v87 0x5d4e04 / v95 0x576030.
//
// packet-audit:fname CITC::OnNormalItemResult#CancelSaleItemDone
type MtsResultCancelSaleItemDone struct {
	mode byte
}

func NewMtsResultCancelSaleItemDone(mode byte) MtsResultCancelSaleItemDone {
	return MtsResultCancelSaleItemDone{mode: mode}
}

func (m MtsResultCancelSaleItemDone) Mode() byte        { return m.mode }
func (m MtsResultCancelSaleItemDone) Operation() string { return MtsOperationWriter }
func (m MtsResultCancelSaleItemDone) String() string {
	return fmt.Sprintf("mts cancel sale item done mode [%d]", m.mode)
}

func (m MtsResultCancelSaleItemDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x25); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultCancelSaleItemDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultSetZzimDone — the 0x29 SetZzimDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a4dfc / v84 0x5b52ec / v87 0x5d4eef / v95 0x576140.
//
// packet-audit:fname CITC::OnNormalItemResult#SetZzimDone
type MtsResultSetZzimDone struct {
	mode byte
}

func NewMtsResultSetZzimDone(mode byte) MtsResultSetZzimDone {
	return MtsResultSetZzimDone{mode: mode}
}

func (m MtsResultSetZzimDone) Mode() byte        { return m.mode }
func (m MtsResultSetZzimDone) Operation() string { return MtsOperationWriter }
func (m MtsResultSetZzimDone) String() string {
	return fmt.Sprintf("mts set zzim done mode [%d]", m.mode)
}

func (m MtsResultSetZzimDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x29); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultSetZzimDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultSetZzimFailed — the 0x2A SetZzimFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a4e31 / v84 0x5b5321 / v87 0x5d4f24 / v95 0x576180.
//
// packet-audit:fname CITC::OnNormalItemResult#SetZzimFailed
type MtsResultSetZzimFailed struct {
	mode byte
}

func NewMtsResultSetZzimFailed(mode byte) MtsResultSetZzimFailed {
	return MtsResultSetZzimFailed{mode: mode}
}

func (m MtsResultSetZzimFailed) Mode() byte        { return m.mode }
func (m MtsResultSetZzimFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultSetZzimFailed) String() string {
	return fmt.Sprintf("mts set zzim failed mode [%d]", m.mode)
}

func (m MtsResultSetZzimFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x2A); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultSetZzimFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultDeleteZzimDone — the 0x2B DeleteZzimDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a4e66 / v84 0x5b5356 / v87 0x5d4f59 / v95 0x5761c0.
//
// packet-audit:fname CITC::OnNormalItemResult#DeleteZzimDone
type MtsResultDeleteZzimDone struct {
	mode byte
}

func NewMtsResultDeleteZzimDone(mode byte) MtsResultDeleteZzimDone {
	return MtsResultDeleteZzimDone{mode: mode}
}

func (m MtsResultDeleteZzimDone) Mode() byte        { return m.mode }
func (m MtsResultDeleteZzimDone) Operation() string { return MtsOperationWriter }
func (m MtsResultDeleteZzimDone) String() string {
	return fmt.Sprintf("mts delete zzim done mode [%d]", m.mode)
}

func (m MtsResultDeleteZzimDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x2B); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultDeleteZzimDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultDeleteZzimFailed — the 0x2C DeleteZzimFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a4e91 / v84 0x5b5381 / v87 0x5d4f84 / v95 0x5761f0.
//
// packet-audit:fname CITC::OnNormalItemResult#DeleteZzimFailed
type MtsResultDeleteZzimFailed struct {
	mode byte
}

func NewMtsResultDeleteZzimFailed(mode byte) MtsResultDeleteZzimFailed {
	return MtsResultDeleteZzimFailed{mode: mode}
}

func (m MtsResultDeleteZzimFailed) Mode() byte        { return m.mode }
func (m MtsResultDeleteZzimFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultDeleteZzimFailed) String() string {
	return fmt.Sprintf("mts delete zzim failed mode [%d]", m.mode)
}

func (m MtsResultDeleteZzimFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x2C); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultDeleteZzimFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultLoadWishSaleListFailed — the 0x2E LoadWishSaleListFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a4fdc / v84 0x5b54cc / v87 0x5d50cf / v95 0x576230.
//
// packet-audit:fname CITC::OnNormalItemResult#LoadWishSaleListFailed
type MtsResultLoadWishSaleListFailed struct {
	mode byte
}

func NewMtsResultLoadWishSaleListFailed(mode byte) MtsResultLoadWishSaleListFailed {
	return MtsResultLoadWishSaleListFailed{mode: mode}
}

func (m MtsResultLoadWishSaleListFailed) Mode() byte        { return m.mode }
func (m MtsResultLoadWishSaleListFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultLoadWishSaleListFailed) String() string {
	return fmt.Sprintf("mts load wish sale list failed mode [%d]", m.mode)
}

func (m MtsResultLoadWishSaleListFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x2E); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultLoadWishSaleListFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultBuyWishDone — the 0x2F BuyWishDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a5011 / v84 0x5b5501 / v87 0x5d5104 / v95 0x576270.
//
// packet-audit:fname CITC::OnNormalItemResult#BuyWishDone
type MtsResultBuyWishDone struct {
	mode byte
}

func NewMtsResultBuyWishDone(mode byte) MtsResultBuyWishDone {
	return MtsResultBuyWishDone{mode: mode}
}

func (m MtsResultBuyWishDone) Mode() byte        { return m.mode }
func (m MtsResultBuyWishDone) Operation() string { return MtsOperationWriter }
func (m MtsResultBuyWishDone) String() string {
	return fmt.Sprintf("mts buy wish done mode [%d]", m.mode)
}

func (m MtsResultBuyWishDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x2F); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultBuyWishDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultBuyWishFailed — the 0x30 BuyWishFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a503c / v84 0x5b552c / v87 0x5d512f / v95 0x5762a0.
//
// packet-audit:fname CITC::OnNormalItemResult#BuyWishFailed
type MtsResultBuyWishFailed struct {
	mode byte
}

func NewMtsResultBuyWishFailed(mode byte) MtsResultBuyWishFailed {
	return MtsResultBuyWishFailed{mode: mode}
}

func (m MtsResultBuyWishFailed) Mode() byte        { return m.mode }
func (m MtsResultBuyWishFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultBuyWishFailed) String() string {
	return fmt.Sprintf("mts buy wish failed mode [%d]", m.mode)
}

func (m MtsResultBuyWishFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x30); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultBuyWishFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultCancelWishDone — the 0x31 CancelWishDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a5071 / v84 0x5b5561 / v87 0x5d5164 / v95 0x5762e0.
//
// packet-audit:fname CITC::OnNormalItemResult#CancelWishDone
type MtsResultCancelWishDone struct {
	mode byte
}

func NewMtsResultCancelWishDone(mode byte) MtsResultCancelWishDone {
	return MtsResultCancelWishDone{mode: mode}
}

func (m MtsResultCancelWishDone) Mode() byte        { return m.mode }
func (m MtsResultCancelWishDone) Operation() string { return MtsOperationWriter }
func (m MtsResultCancelWishDone) String() string {
	return fmt.Sprintf("mts cancel wish done mode [%d]", m.mode)
}

func (m MtsResultCancelWishDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x31); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultCancelWishDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultCancelWishFailed — the 0x32 CancelWishFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a50df / v84 0x5b5596 / v87 0x5d5199 / v95 0x576320.
//
// packet-audit:fname CITC::OnNormalItemResult#CancelWishFailed
type MtsResultCancelWishFailed struct {
	mode byte
}

func NewMtsResultCancelWishFailed(mode byte) MtsResultCancelWishFailed {
	return MtsResultCancelWishFailed{mode: mode}
}

func (m MtsResultCancelWishFailed) Mode() byte        { return m.mode }
func (m MtsResultCancelWishFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultCancelWishFailed) String() string {
	return fmt.Sprintf("mts cancel wish failed mode [%d]", m.mode)
}

func (m MtsResultCancelWishFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x32); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultCancelWishFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultBuyItemDone — the 0x33 BuyItemDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a5114 / v84 0x5b55cb / v87 0x5d51ce / v95 0x576360.
//
// packet-audit:fname CITC::OnNormalItemResult#BuyItemDone
type MtsResultBuyItemDone struct {
	mode byte
}

func NewMtsResultBuyItemDone(mode byte) MtsResultBuyItemDone {
	return MtsResultBuyItemDone{mode: mode}
}

func (m MtsResultBuyItemDone) Mode() byte        { return m.mode }
func (m MtsResultBuyItemDone) Operation() string { return MtsOperationWriter }
func (m MtsResultBuyItemDone) String() string {
	return fmt.Sprintf("mts buy item done mode [%d]", m.mode)
}

func (m MtsResultBuyItemDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x33); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultBuyItemDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultBuyItemFailed — the 0x34 BuyItemFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a513f / v84 0x5b55f6 / v87 0x5d51f9 / v95 0x576390.
//
// packet-audit:fname CITC::OnNormalItemResult#BuyItemFailed
type MtsResultBuyItemFailed struct {
	mode byte
}

func NewMtsResultBuyItemFailed(mode byte) MtsResultBuyItemFailed {
	return MtsResultBuyItemFailed{mode: mode}
}

func (m MtsResultBuyItemFailed) Mode() byte        { return m.mode }
func (m MtsResultBuyItemFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultBuyItemFailed) String() string {
	return fmt.Sprintf("mts buy item failed mode [%d]", m.mode)
}

func (m MtsResultBuyItemFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x34); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultBuyItemFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultBuyZzimItemDone — the 0x35 BuyZzimItemDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a5174 / v84 0x5b562b / v87 0x5d522e / v95 0x5763d0.
//
// packet-audit:fname CITC::OnNormalItemResult#BuyZzimItemDone
type MtsResultBuyZzimItemDone struct {
	mode byte
}

func NewMtsResultBuyZzimItemDone(mode byte) MtsResultBuyZzimItemDone {
	return MtsResultBuyZzimItemDone{mode: mode}
}

func (m MtsResultBuyZzimItemDone) Mode() byte        { return m.mode }
func (m MtsResultBuyZzimItemDone) Operation() string { return MtsOperationWriter }
func (m MtsResultBuyZzimItemDone) String() string {
	return fmt.Sprintf("mts buy zzim item done mode [%d]", m.mode)
}

func (m MtsResultBuyZzimItemDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x35); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultBuyZzimItemDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultBuyZzimItemFailed — the 0x36 BuyZzimItemFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a519f / v84 0x5b5656 / v87 0x5d5259 / v95 0x576400.
//
// packet-audit:fname CITC::OnNormalItemResult#BuyZzimItemFailed
type MtsResultBuyZzimItemFailed struct {
	mode byte
}

func NewMtsResultBuyZzimItemFailed(mode byte) MtsResultBuyZzimItemFailed {
	return MtsResultBuyZzimItemFailed{mode: mode}
}

func (m MtsResultBuyZzimItemFailed) Mode() byte        { return m.mode }
func (m MtsResultBuyZzimItemFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultBuyZzimItemFailed) String() string {
	return fmt.Sprintf("mts buy zzim item failed mode [%d]", m.mode)
}

func (m MtsResultBuyZzimItemFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x36); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultBuyZzimItemFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultRegisterWishItemDone — the 0x37 RegisterWishItemDone arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a51d4 / v84 0x5b568b / v87 0x5d528e / v95 0x576440.
//
// packet-audit:fname CITC::OnNormalItemResult#RegisterWishItemDone
type MtsResultRegisterWishItemDone struct {
	mode byte
}

func NewMtsResultRegisterWishItemDone(mode byte) MtsResultRegisterWishItemDone {
	return MtsResultRegisterWishItemDone{mode: mode}
}

func (m MtsResultRegisterWishItemDone) Mode() byte        { return m.mode }
func (m MtsResultRegisterWishItemDone) Operation() string { return MtsOperationWriter }
func (m MtsResultRegisterWishItemDone) String() string {
	return fmt.Sprintf("mts register wish item done mode [%d]", m.mode)
}

func (m MtsResultRegisterWishItemDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x37); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultRegisterWishItemDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultRegisterWishItemFailed — the 0x38 RegisterWishItemFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a5209 / v84 0x5b56c0 / v87 0x5d52c3 / v95 0x576480.
//
// packet-audit:fname CITC::OnNormalItemResult#RegisterWishItemFailed
type MtsResultRegisterWishItemFailed struct {
	mode byte
}

func NewMtsResultRegisterWishItemFailed(mode byte) MtsResultRegisterWishItemFailed {
	return MtsResultRegisterWishItemFailed{mode: mode}
}

func (m MtsResultRegisterWishItemFailed) Mode() byte        { return m.mode }
func (m MtsResultRegisterWishItemFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultRegisterWishItemFailed) String() string {
	return fmt.Sprintf("mts register wish item failed mode [%d]", m.mode)
}

func (m MtsResultRegisterWishItemFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x38); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultRegisterWishItemFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// MtsResultBidAuctionFailed — the 0x3C BidAuctionFailed arm. The CITC sub-handler is a
// StringPool::GetString + CUtilDlg::Notice with NO CInPacket::Decode* after the
// dispatcher's Decode1(mode); the wire is exactly the mode byte.
// Sub-handler addresses: v83 0x5a5444 / v84 0x5b58fb / v87 0x5d54fe / v95 0x5764c0.
//
// packet-audit:fname CITC::OnNormalItemResult#BidAuctionFailed
type MtsResultBidAuctionFailed struct {
	mode byte
}

func NewMtsResultBidAuctionFailed(mode byte) MtsResultBidAuctionFailed {
	return MtsResultBidAuctionFailed{mode: mode}
}

func (m MtsResultBidAuctionFailed) Mode() byte        { return m.mode }
func (m MtsResultBidAuctionFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultBidAuctionFailed) String() string {
	return fmt.Sprintf("mts bid auction failed mode [%d]", m.mode)
}

func (m MtsResultBidAuctionFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (0x3C); sub-handler reads no further fields
		return w.Bytes()
	}
}

func (m *MtsResultBidAuctionFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// Discrete per-mode "Reason-shape" arms of the CITC::OnNormalItemResult
// dispatcher (MTS_OPERATION). Each arm's CITC sub-handler reads a single
// Decode1 fail-reason byte after the dispatcher's Decode1(mode) — the reason is
// routed to NoticeFailReason (or a reason-keyed StringPool::GetString). No
// further CInPacket::Decode* is performed (the reason==73/65 transfer-field
// re-send branches read NO additional wire bytes). Each mode therefore has its
// OWN discrete struct that writes its (config-resolved) mode byte THEN the
// reason byte — no shared shape struct (task-096: discrete-per-mode rule).
//
// The mode bytes are version-stable across gms_v83 / gms_v84 / gms_v87 /
// gms_v95 (IDA-verified; the case labels are identical and the sub-handler
// bodies are byte-identical in shape). jms_v185 has NO CITC op (registry-absent).
// Per-mode per-version sub-handler addresses are cited in the doc comment above
// each struct (dispatcher: v83 0x5a4311 / v84 0x5b47c8 / v87 0x5d43d0 /
// v95 0x5771d0).

// MtsResultGetItcListFailed — the 0x16 GetITCListFailed arm. The CITC
// sub-handler reads Decode1(reason) -> NoticeFailReason (reason==73 also
// re-sends the transfer-field packet, which reads no further bytes). The wire
// after the dispatcher mode byte is exactly one Decode1 reason byte.
// Sub-handler addresses: v83 0x5a4882 / v84 0x5b4d72 / v87 0x5d4972 / v95 0x575f70.
//
// packet-audit:fname CITC::OnNormalItemResult#GetItcListFailed
type MtsResultGetItcListFailed struct {
	mode   byte
	reason byte
}

func NewMtsResultGetItcListFailed(mode byte, reason byte) MtsResultGetItcListFailed {
	return MtsResultGetItcListFailed{mode: mode, reason: reason}
}

func (m MtsResultGetItcListFailed) Mode() byte        { return m.mode }
func (m MtsResultGetItcListFailed) Reason() byte      { return m.reason }
func (m MtsResultGetItcListFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultGetItcListFailed) String() string {
	return fmt.Sprintf("mts get itc list failed mode [%d] reason [%d]", m.mode, m.reason)
}

func (m MtsResultGetItcListFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte (0x16)
		w.WriteByte(m.reason) // Decode1 fail reason -> NoticeFailReason
		return w.Bytes()
	}
}

func (m *MtsResultGetItcListFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
	}
}

// MtsResultGetSearchItcListFailed — the 0x18 GetSearchITCListFailed arm. The
// CITC sub-handler reads Decode1(reason) -> NoticeFailReason (reason==73 also
// re-sends the transfer-field packet, which reads no further bytes).
// Sub-handler addresses: v83 0x5a49e3 / v84 0x5b4ed3 / v87 0x5d4ad3 / v95 0x575fa0.
//
// packet-audit:fname CITC::OnNormalItemResult#GetSearchItcListFailed
type MtsResultGetSearchItcListFailed struct {
	mode   byte
	reason byte
}

func NewMtsResultGetSearchItcListFailed(mode byte, reason byte) MtsResultGetSearchItcListFailed {
	return MtsResultGetSearchItcListFailed{mode: mode, reason: reason}
}

func (m MtsResultGetSearchItcListFailed) Mode() byte        { return m.mode }
func (m MtsResultGetSearchItcListFailed) Reason() byte      { return m.reason }
func (m MtsResultGetSearchItcListFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultGetSearchItcListFailed) String() string {
	return fmt.Sprintf("mts get search itc list failed mode [%d] reason [%d]", m.mode, m.reason)
}

func (m MtsResultGetSearchItcListFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte (0x18)
		w.WriteByte(m.reason) // Decode1 fail reason -> NoticeFailReason
		return w.Bytes()
	}
}

func (m *MtsResultGetSearchItcListFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
	}
}

// MtsResultSaleCurrentItemToWishFailed — the 0x20 SaleCurrentItemToWishFailed
// arm. The CITC sub-handler reads Decode1(reason) -> reason-keyed
// StringPool::GetString notice (80/82|83/else). The wire after the dispatcher
// mode byte is exactly one Decode1 reason byte.
// Sub-handler addresses: v83 0x5a46f0 / v84 0x5b4be0 / v87 0x5d47c4 / v95 0x575d70.
//
// packet-audit:fname CITC::OnNormalItemResult#SaleCurrentItemToWishFailed
type MtsResultSaleCurrentItemToWishFailed struct {
	mode   byte
	reason byte
}

func NewMtsResultSaleCurrentItemToWishFailed(mode byte, reason byte) MtsResultSaleCurrentItemToWishFailed {
	return MtsResultSaleCurrentItemToWishFailed{mode: mode, reason: reason}
}

func (m MtsResultSaleCurrentItemToWishFailed) Mode() byte        { return m.mode }
func (m MtsResultSaleCurrentItemToWishFailed) Reason() byte      { return m.reason }
func (m MtsResultSaleCurrentItemToWishFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultSaleCurrentItemToWishFailed) String() string {
	return fmt.Sprintf("mts sale current item to wish failed mode [%d] reason [%d]", m.mode, m.reason)
}

func (m MtsResultSaleCurrentItemToWishFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte (0x20)
		w.WriteByte(m.reason) // Decode1 fail reason -> reason-keyed StringPool notice
		return w.Bytes()
	}
}

func (m *MtsResultSaleCurrentItemToWishFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
	}
}

// MtsResultGetUserPurchaseItemFailed — the 0x22 GetUserPurchaseItemFailed arm.
// The CITC sub-handler reads Decode1(reason) -> NoticeFailReason (reason==73
// also re-sends the transfer-field packet, which reads no further bytes).
// Sub-handler addresses: v83 0x5a4c2a / v84 0x5b511a / v87 0x5d4d1a / v95 0x575fd0.
//
// packet-audit:fname CITC::OnNormalItemResult#GetUserPurchaseItemFailed
type MtsResultGetUserPurchaseItemFailed struct {
	mode   byte
	reason byte
}

func NewMtsResultGetUserPurchaseItemFailed(mode byte, reason byte) MtsResultGetUserPurchaseItemFailed {
	return MtsResultGetUserPurchaseItemFailed{mode: mode, reason: reason}
}

func (m MtsResultGetUserPurchaseItemFailed) Mode() byte        { return m.mode }
func (m MtsResultGetUserPurchaseItemFailed) Reason() byte      { return m.reason }
func (m MtsResultGetUserPurchaseItemFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultGetUserPurchaseItemFailed) String() string {
	return fmt.Sprintf("mts get user purchase item failed mode [%d] reason [%d]", m.mode, m.reason)
}

func (m MtsResultGetUserPurchaseItemFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte (0x22)
		w.WriteByte(m.reason) // Decode1 fail reason -> NoticeFailReason
		return w.Bytes()
	}
}

func (m *MtsResultGetUserPurchaseItemFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
	}
}

// MtsResultGetUserSaleItemFailed — the 0x24 GetUserSaleItemFailed arm. The CITC
// sub-handler reads Decode1(reason) -> NoticeFailReason (reason==73 also
// re-sends the transfer-field packet, which reads no further bytes).
// Sub-handler addresses: v83 0x5a4ce7 / v84 0x5b51d7 / v87 0x5d4dd7 / v95 0x576000.
//
// packet-audit:fname CITC::OnNormalItemResult#GetUserSaleItemFailed
type MtsResultGetUserSaleItemFailed struct {
	mode   byte
	reason byte
}

func NewMtsResultGetUserSaleItemFailed(mode byte, reason byte) MtsResultGetUserSaleItemFailed {
	return MtsResultGetUserSaleItemFailed{mode: mode, reason: reason}
}

func (m MtsResultGetUserSaleItemFailed) Mode() byte        { return m.mode }
func (m MtsResultGetUserSaleItemFailed) Reason() byte      { return m.reason }
func (m MtsResultGetUserSaleItemFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultGetUserSaleItemFailed) String() string {
	return fmt.Sprintf("mts get user sale item failed mode [%d] reason [%d]", m.mode, m.reason)
}

func (m MtsResultGetUserSaleItemFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte (0x24)
		w.WriteByte(m.reason) // Decode1 fail reason -> NoticeFailReason
		return w.Bytes()
	}
}

func (m *MtsResultGetUserSaleItemFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
	}
}

// MtsResultCancelSaleItemFailed — the 0x26 CancelSaleItemFailed arm. The CITC
// sub-handler reads exactly Decode1(reason) -> NoticeFailReason with NO further
// CInPacket::Decode*.
// Sub-handler addresses: v83 0x5a4d49 / v84 0x5b5239 / v87 0x5d4e39 / v95 0x576070.
//
// packet-audit:fname CITC::OnNormalItemResult#CancelSaleItemFailed
type MtsResultCancelSaleItemFailed struct {
	mode   byte
	reason byte
}

func NewMtsResultCancelSaleItemFailed(mode byte, reason byte) MtsResultCancelSaleItemFailed {
	return MtsResultCancelSaleItemFailed{mode: mode, reason: reason}
}

func (m MtsResultCancelSaleItemFailed) Mode() byte        { return m.mode }
func (m MtsResultCancelSaleItemFailed) Reason() byte      { return m.reason }
func (m MtsResultCancelSaleItemFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultCancelSaleItemFailed) String() string {
	return fmt.Sprintf("mts cancel sale item failed mode [%d] reason [%d]", m.mode, m.reason)
}

func (m MtsResultCancelSaleItemFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte (0x26)
		w.WriteByte(m.reason) // Decode1 fail reason -> NoticeFailReason
		return w.Bytes()
	}
}

func (m *MtsResultCancelSaleItemFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
	}
}

// MtsResultMoveItcPurchaseItemLtoSFailed — the 0x28 MoveITCPurchaseItemLtoSFailed
// arm. The CITC sub-handler reads Decode1(reason) -> NoticeFailReason
// (reason==65 re-sends the transfer-field packet, which reads no further bytes).
// Sub-handler addresses: v83 0x5a4dcf / v84 0x5b52bf / v87 0x5d4ec2 / v95 0x576110.
//
// packet-audit:fname CITC::OnNormalItemResult#MoveItcPurchaseItemLtoSFailed
type MtsResultMoveItcPurchaseItemLtoSFailed struct {
	mode   byte
	reason byte
}

func NewMtsResultMoveItcPurchaseItemLtoSFailed(mode byte, reason byte) MtsResultMoveItcPurchaseItemLtoSFailed {
	return MtsResultMoveItcPurchaseItemLtoSFailed{mode: mode, reason: reason}
}

func (m MtsResultMoveItcPurchaseItemLtoSFailed) Mode() byte        { return m.mode }
func (m MtsResultMoveItcPurchaseItemLtoSFailed) Reason() byte      { return m.reason }
func (m MtsResultMoveItcPurchaseItemLtoSFailed) Operation() string { return MtsOperationWriter }
func (m MtsResultMoveItcPurchaseItemLtoSFailed) String() string {
	return fmt.Sprintf("mts move itc purchase item ltos failed mode [%d] reason [%d]", m.mode, m.reason)
}

func (m MtsResultMoveItcPurchaseItemLtoSFailed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)   // dispatcher mode byte (0x28)
		w.WriteByte(m.reason) // Decode1 fail reason -> NoticeFailReason
		return w.Bytes()
	}
}

func (m *MtsResultMoveItcPurchaseItemLtoSFailed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.reason = r.ReadByte()
	}
}

// Discrete per-mode "TwoInts-shape" arms of the CITC::OnNormalItemResult
// dispatcher (MTS_OPERATION). Each arm's CITC sub-handler reads exactly two
// Decode4 ints after the dispatcher's Decode1(mode). The downstream use of the
// two ints differs per arm (tab/selectedNo vs notice counts) but the wire read
// order is identical: Decode4 then Decode4. Each mode therefore has its OWN
// discrete struct that writes its (config-resolved) mode byte THEN the two ints — no shared shape struct (task-096: discrete-per-mode rule).
//
// The mode bytes are version-stable across gms_v83 / gms_v84 / gms_v87 /
// gms_v95 (IDA-verified; identical Decode4 then Decode4 read order). jms_v185
// has NO CITC op (registry-absent). Per-mode per-version sub-handler addresses
// are cited above each struct (dispatcher: v83 0x5a4311 / v84 0x5b47c8 /
// v87 0x5d43d0 / v95 0x5771d0).

// MtsResultMoveItcPurchaseItemLtoSDone — the 0x27 MoveITCPurchaseItemLtoSDone
// arm. The CITC sub-handler reads Decode4(tab) -> CCtrlTab::SetTab(tab-1) and
// Decode4(selectedNo). The trailing m_bITCRequestSent=0 store is a member write,
// not a wire read.
// Sub-handler addresses: v83 0x5a4d68 / v84 0x5b5258 / v87 0x5d4e58 / v95 0x5760a0.
//
// packet-audit:fname CITC::OnNormalItemResult#MoveItcPurchaseItemLtoSDone
type MtsResultMoveItcPurchaseItemLtoSDone struct {
	mode       byte
	tab        uint32
	selectedNo uint32
}

func NewMtsResultMoveItcPurchaseItemLtoSDone(mode byte, tab uint32, selectedNo uint32) MtsResultMoveItcPurchaseItemLtoSDone {
	return MtsResultMoveItcPurchaseItemLtoSDone{mode: mode, tab: tab, selectedNo: selectedNo}
}

func (m MtsResultMoveItcPurchaseItemLtoSDone) Mode() byte         { return m.mode }
func (m MtsResultMoveItcPurchaseItemLtoSDone) Tab() uint32        { return m.tab }
func (m MtsResultMoveItcPurchaseItemLtoSDone) SelectedNo() uint32 { return m.selectedNo }
func (m MtsResultMoveItcPurchaseItemLtoSDone) Operation() string  { return MtsOperationWriter }
func (m MtsResultMoveItcPurchaseItemLtoSDone) String() string {
	return fmt.Sprintf("mts move itc purchase item ltos done mode [%d] tab [%d] selectedNo [%d]", m.mode, m.tab, m.selectedNo)
}

func (m MtsResultMoveItcPurchaseItemLtoSDone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)      // dispatcher mode byte (0x27)
		w.WriteInt(m.tab)        // Decode4 tab (-> CCtrlTab::SetTab(tab-1))
		w.WriteInt(m.selectedNo) // Decode4 selectedNo
		return w.Bytes()
	}
}

func (m *MtsResultMoveItcPurchaseItemLtoSDone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.tab = r.ReadUint32()
		m.selectedNo = r.ReadUint32()
	}
}

// MtsResultNotifyCancelWishResult — the 0x3D NotifyCancelWishResult arm. The
// CITC sub-handler reads Decode4(countA) and Decode4(countB); each >0 guards a
// StringPool notice. The wire read order is Decode4 then Decode4.
// Sub-handler addresses: v83 0x5a523e / v84 0x5b56f5 / v87 0x5d52f8 / v95 0x576f00.
//
// packet-audit:fname CITC::OnNormalItemResult#NotifyCancelWishResult
type MtsResultNotifyCancelWishResult struct {
	mode   byte
	countA uint32
	countB uint32
}

func NewMtsResultNotifyCancelWishResult(mode byte, countA uint32, countB uint32) MtsResultNotifyCancelWishResult {
	return MtsResultNotifyCancelWishResult{mode: mode, countA: countA, countB: countB}
}

func (m MtsResultNotifyCancelWishResult) Mode() byte        { return m.mode }
func (m MtsResultNotifyCancelWishResult) CountA() uint32    { return m.countA }
func (m MtsResultNotifyCancelWishResult) CountB() uint32    { return m.countB }
func (m MtsResultNotifyCancelWishResult) Operation() string { return MtsOperationWriter }
func (m MtsResultNotifyCancelWishResult) String() string {
	return fmt.Sprintf("mts notify cancel wish result mode [%d] countA [%d] countB [%d]", m.mode, m.countA, m.countB)
}

func (m MtsResultNotifyCancelWishResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)  // dispatcher mode byte (0x3D)
		w.WriteInt(m.countA) // Decode4 first notice count
		w.WriteInt(m.countB) // Decode4 second notice count
		return w.Bytes()
	}
}

func (m *MtsResultNotifyCancelWishResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.countA = r.ReadUint32()
		m.countB = r.ReadUint32()
	}
}
