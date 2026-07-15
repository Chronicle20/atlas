package merchant

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/merchant/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// ShopScannerResult CWvsContext::OnShopScannerResult
type ShopScannerResultMode = string

const (
	ShopScannerResultModeResult  = ShopScannerResultMode("RESULT")
	ShopScannerResultModeHotList = ShopScannerResultMode("HOT_LIST")
)

// ShopLinkResult CWvsContext::OnShopLinkResult
type ShopLinkResultCode = string

const (
	ShopLinkResultCodeSuccess     = ShopLinkResultCode("SUCCESS")
	ShopLinkResultCodeClosed      = ShopLinkResultCode("CLOSED")
	ShopLinkResultCodeFull        = ShopLinkResultCode("FULL")
	ShopLinkResultCodeBusy        = ShopLinkResultCode("BUSY")
	ShopLinkResultCodeDead        = ShopLinkResultCode("DEAD")
	ShopLinkResultCodeNoTrade     = ShopLinkResultCode("NO_TRADE")
	ShopLinkResultCodeDenied      = ShopLinkResultCode("DENIED")
	ShopLinkResultCodeMaintenance = ShopLinkResultCode("MAINTENANCE")
	ShopLinkResultCodeFMOnly      = ShopLinkResultCode("FM_ONLY")
)

func ShopScannerResultBody(itemId uint32, records []clientbound.ShopScannerRecord) func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ShopScannerResultModeResult, func(mode byte) packet.Encoder {
		return clientbound.NewShopScannerResult(mode, itemId, records)
	})
}

func ShopScannerHotListBody(itemIds []uint32) func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", ShopScannerResultModeHotList, func(mode byte) packet.Encoder {
		return clientbound.NewShopScannerHotList(mode, itemIds)
	})
}

func ShopLinkResultBody(code ShopLinkResultCode) func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", code, func(c byte) packet.Encoder {
		return clientbound.NewShopLinkResult(c)
	})
}
