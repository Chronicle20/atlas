package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	fieldsb "github.com/Chronicle20/atlas/libs/atlas-packet/field/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// ITC_OPERATION operation KEYs. These are the reverse-lookup keys into the
// tenant "operations" table (options["operations"][KEY] -> mode byte). The mode
// bytes themselves are NEVER hard-coded here: the incoming dispatcher mode byte
// is reverse-resolved to one of these KEYs via the config table (mirroring
// isMessengerShopOperation in messenger_operation.go), then dispatched. Verified
// table (template_gms_*_1.json ITC_OPERATION options.operations):
//
//	REGISTER_SALE:2 SALE_CURRENT_ITEM:3 REGISTER_WISH_ENTRY:4 GET_ITC_LIST:5
//	SEARCH_ITC_LIST:6 CANCEL_SALE:7 TAKE_HOME:8 SET_ZZIM:9 DELETE_ZZIM:10
//	VIEW_WISH:11 BUY_WISH:12 CANCEL_WISH:13 BUY:16 BUY_ZZIM:17 REGISTER_AUCTION:18
//	PLACE_BID:19 BUY_AUCTION_IMM:20
const (
	ItcOperationRegisterSale      = "REGISTER_SALE"
	ItcOperationSaleCurrentItem   = "SALE_CURRENT_ITEM"
	ItcOperationRegisterWishEntry = "REGISTER_WISH_ENTRY"
	ItcOperationGetItcList        = "GET_ITC_LIST"
	ItcOperationSearchItcList     = "SEARCH_ITC_LIST"
	ItcOperationCancelSale        = "CANCEL_SALE"
	ItcOperationTakeHome          = "TAKE_HOME"
	ItcOperationSetZzim           = "SET_ZZIM"
	ItcOperationDeleteZzim        = "DELETE_ZZIM"
	ItcOperationViewWish          = "VIEW_WISH"
	ItcOperationBuyWish           = "BUY_WISH"
	ItcOperationCancelWish        = "CANCEL_WISH"
	ItcOperationBuy               = "BUY"
	ItcOperationBuyZzim           = "BUY_ZZIM"
	ItcOperationRegisterAuction   = "REGISTER_AUCTION"
	ItcOperationPlaceBid          = "PLACE_BID"
	ItcOperationBuyAuctionImm     = "BUY_AUCTION_IMM"
)

// itcOperationKeys is the full set of routable KEYs. The dispatcher reverse-
// resolves the incoming mode byte against the tenant table for each of these and
// dispatches to the first match.
var itcOperationKeys = []string{
	ItcOperationRegisterSale,
	ItcOperationSaleCurrentItem,
	ItcOperationRegisterWishEntry,
	ItcOperationGetItcList,
	ItcOperationSearchItcList,
	ItcOperationCancelSale,
	ItcOperationTakeHome,
	ItcOperationSetZzim,
	ItcOperationDeleteZzim,
	ItcOperationViewWish,
	ItcOperationBuyWish,
	ItcOperationCancelWish,
	ItcOperationBuy,
	ItcOperationBuyZzim,
	ItcOperationRegisterAuction,
	ItcOperationPlaceBid,
	ItcOperationBuyAuctionImm,
}

// resolveItcOperationKey reverse-resolves a dispatcher mode byte to its
// operation KEY via the tenant "operations" table (options["operations"]). It is
// the inverse of the config-driven mode resolution used by the clientbound
// writers (WithResolvedCode) and mirrors isMessengerShopOperation's forward
// lookup — NO mode byte is hard-coded. Returns ("", false) when the byte does
// not map to any configured KEY (an unrouted/unknown mode).
func resolveItcOperationKey(l logrus.FieldLogger) func(options map[string]interface{}, mode byte) (string, bool) {
	return func(options map[string]interface{}, mode byte) (string, bool) {
		genericCodes, ok := options["operations"]
		if !ok {
			l.Errorf("ITC_OPERATION has no configured operations table.")
			return "", false
		}
		codes, ok := genericCodes.(map[string]interface{})
		if !ok {
			l.Errorf("ITC_OPERATION operations table is malformed.")
			return "", false
		}
		for _, key := range itcOperationKeys {
			res, ok := codes[key].(float64)
			if !ok {
				continue
			}
			if byte(res) == mode {
				return key, true
			}
		}
		return "", false
	}
}

// ItcOperationHandleFunc is the ITC_OPERATION mode dispatcher. It decodes the
// leading mode byte, reverse-resolves it to an operation KEY against the tenant
// "operations" table, then routes to the per-arm handler. The per-arm bodies are
// filled in by sibling tasks; this foundation routes every configured KEY to a
// clearly-logged seam so an unimplemented arm is a graceful no-op (never a crash)
// and an unconfigured/unknown mode byte is logged rather than silently dropped.
func ItcOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := fieldsb.ItcOperation{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		key, ok := resolveItcOperationKey(l)(readerOptions, p.Mode())
		if !ok {
			l.Warnf("Character [%d] sent ITC_OPERATION with unconfigured/unknown mode byte [%d].", s.CharacterId(), p.Mode())
			return
		}

		switch key {
		case ItcOperationRegisterSale,
			ItcOperationSaleCurrentItem,
			ItcOperationRegisterWishEntry,
			ItcOperationGetItcList,
			ItcOperationSearchItcList,
			ItcOperationCancelSale,
			ItcOperationTakeHome,
			ItcOperationSetZzim,
			ItcOperationDeleteZzim,
			ItcOperationViewWish,
			ItcOperationBuyWish,
			ItcOperationCancelWish,
			ItcOperationBuy,
			ItcOperationBuyZzim,
			ItcOperationRegisterAuction,
			ItcOperationPlaceBid,
			ItcOperationBuyAuctionImm:
			// Routed-but-unimplemented seam. Sibling arm tasks decode the
			// matching fieldsb.ItcOperation* body (via r/readerOptions) and emit
			// the corresponding COMMAND_TOPIC_MTS command + clientbound result.
			l.Infof("Character [%d] sent routed-but-unimplemented ITC_OPERATION [%s] (mode [%d]).", s.CharacterId(), key, p.Mode())
		default:
			l.Warnf("Character [%d] sent ITC_OPERATION with resolved-but-unrouted KEY [%s] (mode [%d]).", s.CharacterId(), key, p.Mode())
		}
	}
}
