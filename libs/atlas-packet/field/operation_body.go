package field

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// MtsOperation result-mode keys (CITC::OnNormalItemResult). Each resolves to the
// per-version byte via the tenant "operations" table (docs/packets/dispatchers/
// mts_operation.yaml). Atlas has no MTS feature emitting these yet; the body
// functions below are wired config-driven (like the other dispatcher families)
// so a future MTS implementation sends the version-correct mode for the chosen
// result. The mode bytes are version-stable across gms_v83/v84/v87/v95
// (IDA-verified); jms has no CITC op.
const (
	MtsOperationGetItcListDone            = "GET_ITC_LIST_DONE"
	MtsOperationGetItcListFailed          = "GET_ITC_LIST_FAILED"
	MtsOperationGetSearchItcListDone      = "GET_SEARCH_ITC_LIST_DONE"
	MtsOperationGetSearchItcListFailed    = "GET_SEARCH_ITC_LIST_FAILED"
	MtsOperationRegisterSaleEntryDone     = "REGISTER_SALE_ENTRY_DONE"
	MtsOperationRegisterSaleEntryFailed   = "REGISTER_SALE_ENTRY_FAILED"
	MtsOperationSaleCurrentItemToWishDone = "SALE_CURRENT_ITEM_TO_WISH_DONE"
	MtsOperationSaleCurrentItemToWishFail = "SALE_CURRENT_ITEM_TO_WISH_FAILED"
	MtsOperationGetUserPurchaseItemDone   = "GET_USER_PURCHASE_ITEM_DONE"
	MtsOperationGetUserPurchaseItemFailed = "GET_USER_PURCHASE_ITEM_FAILED"
	MtsOperationGetUserSaleItemDone       = "GET_USER_SALE_ITEM_DONE"
	MtsOperationGetUserSaleItemFailed     = "GET_USER_SALE_ITEM_FAILED"
	MtsOperationCancelSaleItemDone        = "CANCEL_SALE_ITEM_DONE"
	MtsOperationCancelSaleItemFailed      = "CANCEL_SALE_ITEM_FAILED"
	MtsOperationMoveItcPurchaseItemLtoSDn = "MOVE_ITC_PURCHASE_ITEM_LTOS_DONE"
	MtsOperationMoveItcPurchaseItemLtoSFl = "MOVE_ITC_PURCHASE_ITEM_LTOS_FAILED"
	MtsOperationSetZzimDone               = "SET_ZZIM_DONE"
	MtsOperationSetZzimFailed             = "SET_ZZIM_FAILED"
	MtsOperationDeleteZzimDone            = "DELETE_ZZIM_DONE"
	MtsOperationDeleteZzimFailed          = "DELETE_ZZIM_FAILED"
	MtsOperationLoadWishSaleListDone      = "LOAD_WISH_SALE_LIST_DONE"
	MtsOperationLoadWishSaleListFailed    = "LOAD_WISH_SALE_LIST_FAILED"
	MtsOperationBuyWishDone               = "BUY_WISH_DONE"
	MtsOperationBuyWishFailed             = "BUY_WISH_FAILED"
	MtsOperationCancelWishDone            = "CANCEL_WISH_DONE"
	MtsOperationCancelWishFailed          = "CANCEL_WISH_FAILED"
	MtsOperationBuyItemDone               = "BUY_ITEM_DONE"
	MtsOperationBuyItemFailed             = "BUY_ITEM_FAILED"
	MtsOperationBuyZzimItemDone           = "BUY_ZZIM_ITEM_DONE"
	MtsOperationBuyZzimItemFailed         = "BUY_ZZIM_ITEM_FAILED"
	MtsOperationRegisterWishItemDone      = "REGISTER_WISH_ITEM_DONE"
	MtsOperationRegisterWishItemFailed    = "REGISTER_WISH_ITEM_FAILED"
	MtsOperationBidAuctionFailed          = "BID_AUCTION_FAILED"
	MtsOperationNotifyCancelWishResult    = "NOTIFY_CANCEL_WISH_RESULT"
	MtsOperationSuccessBidInfoResult      = "SUCCESS_BID_INFO_RESULT"
)

// --- Empty-shape arms (notice-only; no wire body after the mode byte) ---------
//
// The CITC sub-handlers for these keys read NOTHING after the dispatcher mode
// byte (they show a StringPool notice and clear m_bITCRequestSent). All resolve
// to the verified clientbound.MtsResultEmpty codec. MtsOperationNoticeBody is the
// keyed helper that constructs the codec for any of these keys; the per-key
// wrappers below give every supported mode a discoverable typed entry point.

// MtsOperationNoticeBody resolves the mode for op (one of the Empty-shape result
// keys) and writes the mode byte only — the verified wire contract for the
// notice-only CITC::OnNormalItemResult arms (clientbound.MtsResultEmpty).
func MtsOperationNoticeBody(op string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", op, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultEmpty(mode)
	})
}

func MtsOperationRegisterSaleEntryDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationRegisterSaleEntryDone)
}

func MtsOperationSaleCurrentItemToWishDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationSaleCurrentItemToWishDone)
}

func MtsOperationCancelSaleItemDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationCancelSaleItemDone)
}

func MtsOperationSetZzimDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationSetZzimDone)
}

func MtsOperationSetZzimFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationSetZzimFailed)
}

func MtsOperationDeleteZzimDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationDeleteZzimDone)
}

func MtsOperationDeleteZzimFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationDeleteZzimFailed)
}

func MtsOperationLoadWishSaleListFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationLoadWishSaleListFailed)
}

func MtsOperationBuyWishDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationBuyWishDone)
}

func MtsOperationBuyWishFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationBuyWishFailed)
}

func MtsOperationCancelWishDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationCancelWishDone)
}

func MtsOperationCancelWishFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationCancelWishFailed)
}

func MtsOperationBuyItemDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationBuyItemDone)
}

func MtsOperationBuyItemFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationBuyItemFailed)
}

func MtsOperationBuyZzimItemDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationBuyZzimItemDone)
}

func MtsOperationBuyZzimItemFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationBuyZzimItemFailed)
}

func MtsOperationRegisterWishItemDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationRegisterWishItemDone)
}

func MtsOperationRegisterWishItemFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationRegisterWishItemFailed)
}

func MtsOperationBidAuctionFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationNoticeBody(MtsOperationBidAuctionFailed)
}

// --- Reason-shape arms (Decode1 fail-reason byte after the mode byte) ----------
//
// All resolve to the verified clientbound.MtsResultReason codec.
// MtsOperationReasonBody is the keyed helper; the per-key wrappers below give
// every supported mode a discoverable typed entry point.

// MtsOperationReasonBody resolves the mode for op (one of the Reason-shape result
// keys) and writes the mode byte THEN the Decode1 fail-reason byte
// (clientbound.MtsResultReason).
func MtsOperationReasonBody(op string, reason byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", op, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultReason(mode, reason)
	})
}

func MtsOperationGetItcListFailedBody(reason byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationReasonBody(MtsOperationGetItcListFailed, reason)
}

func MtsOperationGetSearchItcListFailedBody(reason byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationReasonBody(MtsOperationGetSearchItcListFailed, reason)
}

func MtsOperationSaleCurrentItemToWishFailedBody(reason byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationReasonBody(MtsOperationSaleCurrentItemToWishFail, reason)
}

func MtsOperationGetUserPurchaseItemFailedBody(reason byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationReasonBody(MtsOperationGetUserPurchaseItemFailed, reason)
}

func MtsOperationGetUserSaleItemFailedBody(reason byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationReasonBody(MtsOperationGetUserSaleItemFailed, reason)
}

func MtsOperationCancelSaleItemFailedBody(reason byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationReasonBody(MtsOperationCancelSaleItemFailed, reason)
}

func MtsOperationMoveItcPurchaseItemLtoSFailedBody(reason byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationReasonBody(MtsOperationMoveItcPurchaseItemLtoSFl, reason)
}

// --- TwoInts-shape arms (Decode4 then Decode4 after the mode byte) -------------
//
// Both resolve to the verified clientbound.MtsResultTwoInts codec.
// MtsOperationTwoIntsBody is the keyed helper; the per-key wrappers below give
// every supported mode a discoverable typed entry point with named params.

// MtsOperationTwoIntsBody resolves the mode for op (one of the TwoInts-shape
// result keys) and writes the mode byte THEN two Decode4 ints
// (clientbound.MtsResultTwoInts).
func MtsOperationTwoIntsBody(op string, a uint32, b uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", op, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultTwoInts(mode, a, b)
	})
}

// MtsOperationMoveItcPurchaseItemLtoSDoneBody — 0x27. a = tab (the sub-handler
// adds 1 before CCtrlTab::SetTab), b = selectedNo.
func MtsOperationMoveItcPurchaseItemLtoSDoneBody(tab uint32, selectedNo uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationTwoIntsBody(MtsOperationMoveItcPurchaseItemLtoSDn, tab, selectedNo)
}

// MtsOperationNotifyCancelWishResultBody — 0x3D. a/b are the two notice counts
// (each >0 gates a StringPool notice in the sub-handler).
func MtsOperationNotifyCancelWishResultBody(countA uint32, countB uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return MtsOperationTwoIntsBody(MtsOperationNotifyCancelWishResult, countA, countB)
}

// --- Conditional-tail scalar arms ---------------------------------------------

// MtsOperationRegisterSaleEntryFailedBody — 0x1E. Writes the mode byte, the
// Decode1 fail reason, and — ONLY when reason==0x48 (sale-limit reached) — a
// trailing Decode2 sale-limit short (clientbound.MtsResultRegisterSaleEntryFailed,
// which fixes its own 0x1E mode internally; the resolved mode is version-stable).
func MtsOperationRegisterSaleEntryFailedBody(reason byte, saleLimit uint16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationRegisterSaleEntryFailed, func(_ byte) packet.Encoder {
		return clientbound.NewMtsResultRegisterSaleEntryFailed(reason, saleLimit)
	})
}

// MtsOperationSuccessBidInfoResultBody — 0x3E. Writes the mode byte, the Decode1
// soldFlag, the Decode4 itemId, and — ONLY when itemId>0 — a Decode4 price and an
// 8-byte FILETIME contract date (clientbound.MtsResultSuccessBidInfo, which fixes
// its own 0x3E mode internally; the resolved mode is version-stable).
func MtsOperationSuccessBidInfoResultBody(soldFlag byte, itemId uint32, price uint32, contractDate [8]byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationSuccessBidInfoResult, func(_ byte) packet.Encoder {
		return clientbound.NewMtsResultSuccessBidInfo(soldFlag, itemId, price, contractDate)
	})
}

// --- List / item-blob arms (count then count×ITCITEM) -------------------------
//
// Each resolves to its verified list codec. The codecs fix their own mode byte
// internally (version-stable); WithResolvedCode keeps the family's config-driven
// contract consistent.

// MtsOperationGetItcListDoneBody — 0x15. clientbound.MtsResultGetItcListDone.
func MtsOperationGetItcListDoneBody(categoryItemCnt uint32, category uint32, subCategory uint32, page uint32, sortType byte, sortColumn byte, items []clientbound.MtsItem, requestSent byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationGetItcListDone, func(_ byte) packet.Encoder {
		return clientbound.NewMtsResultGetItcListDone(categoryItemCnt, category, subCategory, page, sortType, sortColumn, items, requestSent)
	})
}

// MtsOperationGetSearchItcListDoneBody — 0x17.
// clientbound.MtsResultGetSearchItcListDone.
func MtsOperationGetSearchItcListDoneBody(categoryItemCnt uint32, category uint32, subCategory uint32, page uint32, items []clientbound.MtsItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationGetSearchItcListDone, func(_ byte) packet.Encoder {
		return clientbound.NewMtsResultGetSearchItcListDone(categoryItemCnt, category, subCategory, page, items)
	})
}

// MtsOperationGetUserPurchaseItemDoneBody — 0x21.
// clientbound.MtsResultGetUserPurchaseItemDone.
func MtsOperationGetUserPurchaseItemDoneBody(items []clientbound.MtsItem, limitedCount uint32, requestSent byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationGetUserPurchaseItemDone, func(_ byte) packet.Encoder {
		return clientbound.NewMtsResultGetUserPurchaseItemDone(items, limitedCount, requestSent)
	})
}

// MtsOperationGetUserSaleItemDoneBody — 0x23.
// clientbound.MtsResultGetUserSaleItemDone.
func MtsOperationGetUserSaleItemDoneBody(items []clientbound.MtsItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationGetUserSaleItemDone, func(_ byte) packet.Encoder {
		return clientbound.NewMtsResultGetUserSaleItemDone(items)
	})
}

// MtsOperationLoadWishSaleListDoneBody — 0x2D.
// clientbound.MtsResultLoadWishSaleListDone.
func MtsOperationLoadWishSaleListDoneBody(items []clientbound.MtsItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationLoadWishSaleListDone, func(_ byte) packet.Encoder {
		return clientbound.NewMtsResultLoadWishSaleListDone(items)
	})
}

// MtsOperationNewItem constructs one ITCITEM entry for the list-arm body
// functions above (clientbound.MtsItem). It re-exports the verified MtsItem
// constructor so callers in the field package do not need to import clientbound
// directly. The model.Asset blob is the embedded GW_ItemSlotBase item.
func MtsOperationNewItem(item model.Asset, itcSn uint32, price uint32, contractFee uint32, contractFeeTx string, rollbackUsage string, dateExpired [8]byte, userId string, gameId string, comment string, bidCount uint32, bidRange uint32, bidPrice uint32, minPrice uint32, maxPrice uint32, unitPrice uint32, processStatus uint16) clientbound.MtsItem {
	return clientbound.NewMtsItem(item, itcSn, price, contractFee, contractFeeTx, rollbackUsage, dateExpired, userId, gameId, comment, bidCount, bidRange, bidPrice, minPrice, maxPrice, unitPrice, processStatus)
}
