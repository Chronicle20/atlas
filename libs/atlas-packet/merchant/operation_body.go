package merchant

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	"github.com/Chronicle20/atlas-packet/merchant/clientbound"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

type HiredMerchantOperationMode = string

const (
	// HiredMerchantOperation CWvsContext::OnEntrustedShopCheckResult

	HiredMerchantOperationModeOpenShop                            HiredMerchantOperationMode = "OPEN_SHOP"                                 // 7
	HiredMerchantOperationModeErrorUnknown                        HiredMerchantOperationMode = "ERROR_UNKNOWN"                             // 8
	HiredMerchantOperationModeErrorRetrieveFromFredrick           HiredMerchantOperationMode = "ERROR_RETRIEVE_FROM_FREDRICK"              // 9
	HiredMerchantOperationModeErrorAnotherCharacterIsUsingTheItem HiredMerchantOperationMode = "ERROR_ANOTHER_CHARACTER_IS_USING_THE_ITEM" // 10
	HiredMerchantOperationModeErrorUnableToOpenTheStore           HiredMerchantOperationMode = "ERROR_UNABLE_TO_OPEN_THE_STORE"            // 1
	HiredMerchantOperationModeShopSearch                          HiredMerchantOperationMode = "SHOP_SEARCH"                               // 13
	HiredMerchantOperationModeShopRename                          HiredMerchantOperationMode = "SHOP_RENAME"                               // 14
	HiredMerchantOperationModeErrorRetrieveFromFredrick2          HiredMerchantOperationMode = "ERROR_RETRIEVE_FROM_FREDRICK_2"            // 15
	HiredMerchantOperationModeRemoteShopWarp                      HiredMerchantOperationMode = "REMOTE_SHOP_WARP"                          // 16
	HiredMerchantOperationModeConfirmManage                       HiredMerchantOperationMode = "CONFIRM_MANAGE"                            // 17
	HiredMerchantOperationModeFreeFormNotice                      HiredMerchantOperationMode = "FREE_FORM_NOTICE"                          // 18
)

func HiredMerchantOperationOpenShopBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", HiredMerchantOperationModeOpenShop, func(mode byte) packet.Encoder {
		return clientbound.NewOpenShop(mode)
	})
}

func HiredMerchantOperationErrorRetrieveFromFredrickBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", HiredMerchantOperationModeErrorRetrieveFromFredrick, func(mode byte) packet.Encoder {
		return clientbound.NewMerchantErrorSimple(mode)
	})
}

func HiredMerchantOperationErrorAnotherCharacterIsUsingTheItemBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", HiredMerchantOperationModeErrorAnotherCharacterIsUsingTheItem, func(mode byte) packet.Encoder {
		return clientbound.NewMerchantErrorSimple(mode)
	})
}

func HiredMerchantOperationErrorUnableToOpenTheStoreBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", HiredMerchantOperationModeErrorUnableToOpenTheStore, func(mode byte) packet.Encoder {
		return clientbound.NewMerchantErrorSimple(mode)
	})
}

func HiredMerchantOperationShopSearchBody(shopId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", HiredMerchantOperationModeShopSearch, func(mode byte) packet.Encoder {
		return clientbound.NewShopSearch(mode, shopId)
	})
}

func HiredMerchantOperationShopRenameBody(success bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", HiredMerchantOperationModeShopRename, func(mode byte) packet.Encoder {
		return clientbound.NewShopRename(mode, success)
	})
}

func HiredMerchantOperationErrorRetrieveFromFredrick2Body() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", HiredMerchantOperationModeErrorRetrieveFromFredrick2, func(mode byte) packet.Encoder {
		return clientbound.NewMerchantErrorSimple(mode)
	})
}

func HiredMerchantOperationRemoteShopWarpBody(shopId uint32, channelId byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", HiredMerchantOperationModeRemoteShopWarp, func(mode byte) packet.Encoder {
		return clientbound.NewRemoteShopWarp(mode, shopId, channelId)
	})
}

func HiredMerchantOperationRemoteShopWarpErrorBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return HiredMerchantOperationRemoteShopWarpBody(0, 255)
}

// ConfirmManage - TODO This immediately triggers a PLAYER_INTERACTION after retrieving birthday from the client, need to confirm variable naming
func HiredMerchantOperationConfirmManageBody(shopId uint32, position uint16, serialNumber uint64) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", HiredMerchantOperationModeConfirmManage, func(mode byte) packet.Encoder {
		return clientbound.NewConfirmManage(mode, shopId, position, serialNumber)
	})
}

func HiredMerchantOperationFreeFormNoticeBody(message string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", HiredMerchantOperationModeFreeFormNotice, func(mode byte) packet.Encoder {
		return clientbound.NewFreeFormNotice(mode, message)
	})
}
