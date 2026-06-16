package writer

import (
	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// MtsOperation result-mode keys (CITC::OnNormalItemResult). Each resolves to the
// per-version byte via the tenant "operations" table (docs/packets/dispatchers/
// mts_operation.yaml). Atlas has no MTS feature emitting these yet; the codec is
// wired config-driven (like the other dispatcher families) so a future MTS
// implementation sends the version-correct mode for the chosen result.
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

// MtsOperationBody resolves the MTS result mode for op from the tenant
// "operations" table and writes the OP-MODE-PREFIX (the leading mode byte that
// CITC::OnNormalItemResult switch-dispatches on).
func MtsOperationBody(op string) packet.Encode {
	return atlas_packet.WithResolvedCode("operations", op, func(mode byte) packet.Encoder {
		return fieldcb.NewMtsOperation(mode)
	})
}
