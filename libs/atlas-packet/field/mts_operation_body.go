package field

import (
	"context"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
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
// byte (they show a StringPool notice and clear m_bITCRequestSent). Each mode
// has its OWN discrete clientbound struct; the body function resolves the
// per-version mode byte via WithResolvedCode (keyed by the fixed operation KEY)
// and passes it into the struct constructor — the config-driven contract shared
// by the other dispatcher families (npc/storage/cash).

func MtsOperationRegisterSaleEntryDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationRegisterSaleEntryDone, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultRegisterSaleEntryDone(mode)
	})
}

func MtsOperationSaleCurrentItemToWishDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationSaleCurrentItemToWishDone, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultSaleCurrentItemToWishDone(mode)
	})
}

func MtsOperationCancelSaleItemDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationCancelSaleItemDone, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultCancelSaleItemDone(mode)
	})
}

func MtsOperationSetZzimDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationSetZzimDone, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultSetZzimDone(mode)
	})
}

func MtsOperationSetZzimFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationSetZzimFailed, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultSetZzimFailed(mode)
	})
}

func MtsOperationDeleteZzimDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationDeleteZzimDone, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultDeleteZzimDone(mode)
	})
}

func MtsOperationDeleteZzimFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationDeleteZzimFailed, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultDeleteZzimFailed(mode)
	})
}

func MtsOperationLoadWishSaleListFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationLoadWishSaleListFailed, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultLoadWishSaleListFailed(mode)
	})
}

func MtsOperationBuyWishDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationBuyWishDone, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultBuyWishDone(mode)
	})
}

func MtsOperationBuyWishFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationBuyWishFailed, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultBuyWishFailed(mode)
	})
}

func MtsOperationCancelWishDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationCancelWishDone, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultCancelWishDone(mode)
	})
}

func MtsOperationCancelWishFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationCancelWishFailed, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultCancelWishFailed(mode)
	})
}

func MtsOperationBuyItemDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationBuyItemDone, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultBuyItemDone(mode)
	})
}

func MtsOperationBuyItemFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationBuyItemFailed, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultBuyItemFailed(mode)
	})
}

func MtsOperationBuyZzimItemDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationBuyZzimItemDone, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultBuyZzimItemDone(mode)
	})
}

func MtsOperationBuyZzimItemFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationBuyZzimItemFailed, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultBuyZzimItemFailed(mode)
	})
}

func MtsOperationRegisterWishItemDoneBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationRegisterWishItemDone, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultRegisterWishItemDone(mode)
	})
}

func MtsOperationRegisterWishItemFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationRegisterWishItemFailed, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultRegisterWishItemFailed(mode)
	})
}

func MtsOperationBidAuctionFailedBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationBidAuctionFailed, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultBidAuctionFailed(mode)
	})
}

// --- Reason-shape arms (Decode1 fail-reason byte after the mode byte) ----------
//
// Each mode has its OWN discrete clientbound struct that writes the mode byte
// THEN the Decode1 fail-reason byte; the body function resolves the per-version
// mode byte via WithResolvedCode (keyed by the fixed operation KEY) and passes
// it into the constructor (config-driven, like npc/storage/cash).

func MtsOperationGetItcListFailedBody(reason byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationGetItcListFailed, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultGetItcListFailed(mode, reason)
	})
}

func MtsOperationGetSearchItcListFailedBody(reason byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationGetSearchItcListFailed, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultGetSearchItcListFailed(mode, reason)
	})
}

func MtsOperationSaleCurrentItemToWishFailedBody(reason byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationSaleCurrentItemToWishFail, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultSaleCurrentItemToWishFailed(mode, reason)
	})
}

func MtsOperationGetUserPurchaseItemFailedBody(reason byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationGetUserPurchaseItemFailed, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultGetUserPurchaseItemFailed(mode, reason)
	})
}

func MtsOperationGetUserSaleItemFailedBody(reason byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationGetUserSaleItemFailed, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultGetUserSaleItemFailed(mode, reason)
	})
}

func MtsOperationCancelSaleItemFailedBody(reason byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationCancelSaleItemFailed, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultCancelSaleItemFailed(mode, reason)
	})
}

func MtsOperationMoveItcPurchaseItemLtoSFailedBody(reason byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationMoveItcPurchaseItemLtoSFl, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultMoveItcPurchaseItemLtoSFailed(mode, reason)
	})
}

// --- TwoInts-shape arms (Decode4 then Decode4 after the mode byte) -------------
//
// Each mode has its OWN discrete clientbound struct that writes the mode byte
// THEN two Decode4 ints; the body function resolves the per-version mode byte
// via WithResolvedCode (keyed by the fixed operation KEY) and passes it into the
// constructor (config-driven, like npc/storage/cash).

// MtsOperationMoveItcPurchaseItemLtoSDoneBody — 0x27. tab (the sub-handler adds
// 1 before CCtrlTab::SetTab) and selectedNo.
func MtsOperationMoveItcPurchaseItemLtoSDoneBody(tab uint32, selectedNo uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationMoveItcPurchaseItemLtoSDn, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultMoveItcPurchaseItemLtoSDone(mode, tab, selectedNo)
	})
}

// MtsOperationNotifyCancelWishResultBody — 0x3D. countA/countB are the two notice
// counts (each >0 gates a StringPool notice in the sub-handler).
func MtsOperationNotifyCancelWishResultBody(countA uint32, countB uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationNotifyCancelWishResult, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultNotifyCancelWishResult(mode, countA, countB)
	})
}

// --- Conditional-tail scalar arms ---------------------------------------------

// MtsOperationRegisterSaleEntryFailedBody — 0x1E. Writes the mode byte, the
// Decode1 fail reason, and — ONLY when reason==0x48 (sale-limit reached) — a
// trailing Decode2 sale-limit short (clientbound.MtsResultRegisterSaleEntryFailed;
// the resolved mode byte (0x1E, version-stable) is passed into the constructor).
func MtsOperationRegisterSaleEntryFailedBody(reason byte, saleLimit uint16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationRegisterSaleEntryFailed, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultRegisterSaleEntryFailed(mode, reason, saleLimit)
	})
}

// MtsOperationSuccessBidInfoResultBody — 0x3E. Writes the mode byte, the Decode1
// soldFlag, the Decode4 itemId, and — ONLY when itemId>0 — a Decode4 price and an
// 8-byte FILETIME contract date (clientbound.MtsResultSuccessBidInfo; the
// resolved mode byte (0x3E, version-stable) is passed into the constructor).
func MtsOperationSuccessBidInfoResultBody(soldFlag byte, itemId uint32, price uint32, contractDate [8]byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationSuccessBidInfoResult, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultSuccessBidInfo(mode, soldFlag, itemId, price, contractDate)
	})
}

// --- List / item-blob arms (count then count×ITCITEM) -------------------------
//
// Each resolves to its verified list codec. WithResolvedCode resolves the
// per-version mode byte (version-stable) from the tenant operations table and
// passes it into the codec constructor — the family's config-driven contract.

// MtsOperationGetItcListDoneBody — 0x15. clientbound.MtsResultGetItcListDone.
func MtsOperationGetItcListDoneBody(categoryItemCnt uint32, category uint32, subCategory uint32, page uint32, sortType byte, sortColumn byte, items []clientbound.MtsItem, requestSent byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationGetItcListDone, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultGetItcListDone(mode, categoryItemCnt, category, subCategory, page, sortType, sortColumn, items, requestSent)
	})
}

// MtsOperationGetSearchItcListDoneBody — 0x17.
// clientbound.MtsResultGetSearchItcListDone.
func MtsOperationGetSearchItcListDoneBody(categoryItemCnt uint32, category uint32, subCategory uint32, page uint32, items []clientbound.MtsItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationGetSearchItcListDone, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultGetSearchItcListDone(mode, categoryItemCnt, category, subCategory, page, items)
	})
}

// MtsOperationGetUserPurchaseItemDoneBody — 0x21.
// clientbound.MtsResultGetUserPurchaseItemDone.
func MtsOperationGetUserPurchaseItemDoneBody(items []clientbound.MtsItem, limitedCount uint32, requestSent byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationGetUserPurchaseItemDone, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultGetUserPurchaseItemDone(mode, items, limitedCount, requestSent)
	})
}

// MtsOperationGetUserSaleItemDoneBody — 0x23.
// clientbound.MtsResultGetUserSaleItemDone.
func MtsOperationGetUserSaleItemDoneBody(items []clientbound.MtsItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationGetUserSaleItemDone, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultGetUserSaleItemDone(mode, items)
	})
}

// MtsOperationLoadWishSaleListDoneBody — 0x2D.
// clientbound.MtsResultLoadWishSaleListDone.
func MtsOperationLoadWishSaleListDoneBody(items []clientbound.MtsItem) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", MtsOperationLoadWishSaleListDone, func(mode byte) packet.Encoder {
		return clientbound.NewMtsResultLoadWishSaleListDone(mode, items)
	})
}

// MtsOperationNewItem constructs one ITCITEM entry for the list-arm body
// functions above (clientbound.MtsItem). It re-exports the verified MtsItem
// constructor so callers in the field package do not need to import clientbound
// directly. The model.Asset blob is the embedded GW_ItemSlotBase item.
func MtsOperationNewItem(item model.Asset, itcSn uint32, price uint32, contractFee uint32, contractFeeTx string, rollbackUsage string, dateExpired [8]byte, userId string, gameId string, comment string, bidCount uint32, bidRange uint32, bidPrice uint32, minPrice uint32, maxPrice uint32, unitPrice uint32, processStatusKey string) clientbound.MtsItem {
	return clientbound.NewMtsItem(item, itcSn, price, contractFee, contractFeeTx, rollbackUsage, dateExpired, userId, gameId, comment, bidCount, bidRange, bidPrice, minPrice, maxPrice, unitPrice, processStatusKey)
}
