package writer

import (
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const ServerIP = "ServerIP"

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

func ServerIPBody(l logrus.FieldLogger, tenant tenant.Model) func(ipAddr string, port uint16, clientId uint32) BodyProducer {
	return func(ipAddr string, port uint16, clientId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(ServerIP, string(ServerIPCodeOk), "codes", options))
			w.WriteByte(getCode(l)(ServerIP, string(ServerIPModeOk), "modes", options))
			w.WriteByteArray(ipAsByteArray(ipAddr))
			w.WriteShort(port)
			w.WriteInt(clientId)
			w.WriteByte(0) // bAuthenCode
			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 12) || tenant.Region() == "JMS" {
				w.WriteInt(0) // ulPremiumArgument
			}
			return w.Bytes()
		}
	}
}

func ipAsByteArray(ipAddress string) []byte {
	var ob = make([]byte, 0)
	os := strings.Split(ipAddress, ".")
	for _, x := range os {
		o, err := strconv.ParseUint(x, 10, 8)
		if err == nil {
			ob = append(ob, byte(o))
		}
	}
	return ob
}

func ServerIPBodySimpleError(l logrus.FieldLogger) func(code ServerIPCode) BodyProducer {
	return func(code ServerIPCode) BodyProducer {
		return ServerIPBodyError(l)(code, ServerIPModeOk)
	}
}

func ServerIPBodyError(l logrus.FieldLogger) func(code ServerIPCode, mode ServerIPMode) BodyProducer {
	return func(code ServerIPCode, mode ServerIPMode) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(ServerIP, string(code), "codes", options))
			w.WriteByte(getCode(l)(ServerIP, string(mode), "modes", options))
			return w.Bytes()
		}
	}
}
