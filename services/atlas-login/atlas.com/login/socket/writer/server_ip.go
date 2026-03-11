package writer

import (
	"context"

	loginpkt "github.com/Chronicle20/atlas-packet/login"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)


type ServerIPCode string
type ServerIPMode string

const (
	ServerIPCodeOk                        ServerIPCode = "OK"
	ServerIPCodeIdDeletedOrBlocked        ServerIPCode = "ID_DELETED_OR_BLOCKED"
	ServerIPCodeIncorrectPassword         ServerIPCode = "INCORRECT_PASSWORD"
	ServerIPCodeNotRegisteredId           ServerIPCode = "NOT_REGISTERED_ID"
	ServerIPCodeServerUnderInspection     ServerIPCode = "SERVER_UNDER_INSPECTION"
	ServerIPCodeTooManyConnectionRequests ServerIPCode = "TOO_MANY_CONNECTION_REQUESTS"
	ServerIPCodeAdultChannel              ServerIPCode = "ADULT_CHANNEL"
	ServerIPCodeMasterIP                  ServerIPCode = "MASTER_IP"
	ServerIPCodeWrongGateway              ServerIPCode = "WRONG_GATEWAY"
	ServerIPCodeStillProcessing           ServerIPCode = "STILL_PROCESSING"
	ServerIPCodeAccountVerification       ServerIPCode = "ACCOUNT_VERIFICATION"
	ServerIPCodeMapleEuropeRedirect       ServerIPCode = "MAPLE_EUROPE_REDIRECT"
	ServerIPCodeToTitle                   ServerIPCode = "TO_TITLE"

	ServerIPModeOk                  ServerIPMode = "OK"
	ServerIPModeIncorrectLoginId    ServerIPMode = "INCORRECT_LOGIN_ID"
	ServerIPModeIncorrectFormOfId   ServerIPMode = "INCORRECT_FORM_OF_ID"
	ServerIPModeSevenDayUnverified  ServerIPMode = "SEVEN_DAY_UNVERIFIED"
	ServerIPModeUsedUpGameTime      ServerIPMode = "USED_UP_GAME_TIME"
	ServerIPModeThirtyDayUnverified ServerIPMode = "THIRTY_DAY_UNVERIFIED"
	ServerIPModePCRoomPremium       ServerIPMode = "PC_ROOM_PREMIUM"
)

func ServerIPBody(ipAddr string, port uint16, clientId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			code := getCode(l)(loginpkt.ServerIPWriter, string(ServerIPCodeOk), "codes", options)
			mode := getCode(l)(loginpkt.ServerIPWriter, string(ServerIPModeOk), "modes", options)
			return loginpkt.NewServerIP(code, mode, ipAddr, port, clientId).Encode(l, ctx)(options)
		}
	}
}

func ServerIPBodySimpleError(code ServerIPCode) packet.Encode {
	return ServerIPBodyError(code, ServerIPModeOk)
}

func ServerIPBodyError(code ServerIPCode, mode ServerIPMode) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			codeResolved := getCode(l)(loginpkt.ServerIPWriter, string(code), "codes", options)
			modeResolved := getCode(l)(loginpkt.ServerIPWriter, string(mode), "modes", options)
			return loginpkt.NewServerIPError(codeResolved, modeResolved).Encode(l, ctx)(options)
		}
	}
}
