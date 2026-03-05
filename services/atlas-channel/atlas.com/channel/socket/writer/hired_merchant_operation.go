package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type HiredMerchantOperationMode string

const (
	// HiredMerchantOperation CWvsContext::OnEntrustedShopCheckResult
	HiredMerchantOperation = "HiredMerchantOperation"

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

func HiredMerchantOperationOpenShopBody(l logrus.FieldLogger) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeOpenShop))
		return w.Bytes()
	}
}

func HiredMerchantOperationErrorRetrieveFromFredrickBody(l logrus.FieldLogger) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeErrorRetrieveFromFredrick))
		return w.Bytes()
	}
}

func HiredMerchantOperationErrorAnotherCharacterIsUsingTheItemBody(l logrus.FieldLogger) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeErrorAnotherCharacterIsUsingTheItem))
		return w.Bytes()
	}
}

func HiredMerchantOperationErrorUnableToOpenTheStoreBody(l logrus.FieldLogger) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeErrorUnableToOpenTheStore))
		return w.Bytes()
	}
}

func HiredMerchantOperationShopSearchBody(l logrus.FieldLogger) func(shopId uint32) BodyProducer {
	return func(shopId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeShopSearch))
			w.WriteInt(shopId)
			return w.Bytes()
		}
	}
}

func HiredMerchantOperationShopRenameBody(l logrus.FieldLogger) func(success bool) BodyProducer {
	return func(success bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeShopRename))
			w.WriteBool(success)
			return w.Bytes()
		}
	}
}

func HiredMerchantOperationErrorRetrieveFromFredrick2Body(l logrus.FieldLogger) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeErrorRetrieveFromFredrick2))
		return w.Bytes()
	}
}

func HiredMerchantOperationRemoteShopWarpBody(l logrus.FieldLogger) func(shopId uint32, channelId byte) BodyProducer {
	return func(shopId uint32, channelId byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeRemoteShopWarp))
			w.WriteInt(shopId)
			w.WriteByte(channelId)
			return w.Bytes()
		}
	}
}

func HiredMerchantOperationRemoteShopWarpErrorBody(l logrus.FieldLogger) BodyProducer {
	return HiredMerchantOperationRemoteShopWarpBody(l)(0, 255)
}

func HiredMerchantOperationConfirmManageBody(l logrus.FieldLogger) func(shopId uint32, position uint16, serialNumber uint64) BodyProducer {
	return func(shopId uint32, position uint16, serialNumber uint64) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			// TODO This immediately triggers a PLAYER_INTERACTION after retrieving birthday from the client, need to confirm variable naming
			w.WriteByte(getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeConfirmManage))
			w.WriteInt(shopId)
			w.WriteShort(position)
			w.WriteLong(serialNumber)
			return w.Bytes()
		}
	}
}

func HiredMerchantOperationFreeFormNoticeBody(l logrus.FieldLogger) func(message string) BodyProducer {
	return func(message string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getHiredMerchantOperationMode(l)(options, HiredMerchantOperationModeFreeFormNotice))
			w.WriteBool(true)
			w.WriteAsciiString(message)
			return w.Bytes()
		}
	}
}

func getHiredMerchantOperationMode(l logrus.FieldLogger) func(options map[string]interface{}, key HiredMerchantOperationMode) byte {
	return func(options map[string]interface{}, key HiredMerchantOperationMode) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		op, ok := codes[string(key)].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}
		return byte(op)
	}
}
