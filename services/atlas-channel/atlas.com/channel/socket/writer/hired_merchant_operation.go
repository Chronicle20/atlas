package writer

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	merchantpkt "github.com/Chronicle20/atlas-packet/merchant"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

type HiredMerchantOperationMode string

const (
	// HiredMerchantOperation CWvsContext::OnEntrustedShopCheckResult

	HiredMerchantOperationModeOpenShop                            = "OPEN_SHOP"                                 // 7
	HiredMerchantOperationModeErrorUnknown                        = "ERROR_UNKNOWN"                             // 8
	HiredMerchantOperationModeErrorRetrieveFromFredrick           = "ERROR_RETRIEVE_FROM_FREDRICK"              // 9
	HiredMerchantOperationModeErrorAnotherCharacterIsUsingTheItem = "ERROR_ANOTHER_CHARACTER_IS_USING_THE_ITEM" // 10
	HiredMerchantOperationModeErrorUnableToOpenTheStore           = "ERROR_UNABLE_TO_OPEN_THE_STORE"            // 1
	HiredMerchantOperationModeShopSearch                          = "SHOP_SEARCH"                               // 13
	HiredMerchantOperationModeShopRename                          = "SHOP_RENAME"                               // 14
	HiredMerchantOperationModeErrorRetrieveFromFredrick2          = "ERROR_RETRIEVE_FROM_FREDRICK_2"            // 15
	HiredMerchantOperationModeRemoteShopWarp                      = "REMOTE_SHOP_WARP"                          // 16
	HiredMerchantOperationModeConfirmManage                       = "CONFIRM_MANAGE"                            // 17
	HiredMerchantOperationModeFreeFormNotice                      = "FREE_FORM_NOTICE"                          // 18
)

func HiredMerchantOperationOpenShopBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeOpenShop)
			return merchantpkt.NewOpenShop(mode).Encode(l, ctx)(options)
		}
	}
}

func HiredMerchantOperationErrorRetrieveFromFredrickBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeErrorRetrieveFromFredrick)
			return merchantpkt.NewMerchantErrorSimple(mode).Encode(l, ctx)(options)
		}
	}
}

func HiredMerchantOperationErrorAnotherCharacterIsUsingTheItemBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeErrorAnotherCharacterIsUsingTheItem)
			return merchantpkt.NewMerchantErrorSimple(mode).Encode(l, ctx)(options)
		}
	}
}

func HiredMerchantOperationErrorUnableToOpenTheStoreBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeErrorUnableToOpenTheStore)
			return merchantpkt.NewMerchantErrorSimple(mode).Encode(l, ctx)(options)
		}
	}
}

func HiredMerchantOperationShopSearchBody(shopId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeShopSearch)
			return merchantpkt.NewShopSearch(mode, shopId).Encode(l, ctx)(options)
		}
	}
}

func HiredMerchantOperationShopRenameBody(success bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeShopRename)
			return merchantpkt.NewShopRename(mode, success).Encode(l, ctx)(options)
		}
	}
}

func HiredMerchantOperationErrorRetrieveFromFredrick2Body() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeErrorRetrieveFromFredrick2)
			return merchantpkt.NewMerchantErrorSimple(mode).Encode(l, ctx)(options)
		}
	}
}

func HiredMerchantOperationRemoteShopWarpBody(shopId uint32, channelId byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeRemoteShopWarp)
			return merchantpkt.NewRemoteShopWarp(mode, shopId, channelId).Encode(l, ctx)(options)
		}
	}
}

func HiredMerchantOperationRemoteShopWarpErrorBody() packet.Encode {
	return HiredMerchantOperationRemoteShopWarpBody(0, 255)
}

func HiredMerchantOperationConfirmManageBody(shopId uint32, position uint16, serialNumber uint64) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			// TODO This immediately triggers a PLAYER_INTERACTION after retrieving birthday from the client, need to confirm variable naming
			mode := getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeConfirmManage)
			return merchantpkt.NewConfirmManage(mode, shopId, position, serialNumber).Encode(l, ctx)(options)
		}
	}
}

func HiredMerchantOperationFreeFormNoticeBody(message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeFreeFormNotice)
			return merchantpkt.NewFreeFormNotice(mode, message).Encode(l, ctx)(options)
		}
	}
}

func getHiredMerchantOperationMode(l logrus.FieldLogger) func(options map[string]interface{}, key HiredMerchantOperationMode) byte {
	return func(options map[string]interface{}, key HiredMerchantOperationMode) byte {
		return atlas_packet.ResolveCode(l, options, "operations", string(key))
	}
}
