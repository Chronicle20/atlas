package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
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

func ServerIPBody(l logrus.FieldLogger) func(ipAddr string, port uint16, clientId uint32) BodyProducer {
	return func(ipAddr string, port uint16, clientId uint32) BodyProducer {
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteByte(getServerIPCode(l)(ServerIPCodeOk, options))
			w.WriteByte(getServerIPMode(l)(ServerIPModeOk, options))
			w.WriteByteArray(ipAsByteArray(ipAddr))
			w.WriteShort(port)
			w.WriteInt(clientId)
			w.WriteByte(0) // bAuthenCode
			w.WriteInt(0)  // ulPremiumArgument
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
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteByte(getServerIPCode(l)(code, options))
			w.WriteByte(getServerIPMode(l)(mode, options))
			return w.Bytes()
		}
	}
}

const serverIPCodeProperty = "codes"

const serverIPModeProperty = "modes"

func getServerIPCode(l logrus.FieldLogger) func(code ServerIPCode, options map[string]interface{}) byte {
	return func(codeString ServerIPCode, options map[string]interface{}) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options[serverIPCodeProperty]; !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", codeString, AddCharacterEntry)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", codeString, AddCharacterEntry)
			return 99
		}

		code, ok := codes[string(codeString)].(float64)
		if !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", codeString, AddCharacterEntry)
			return 99
		}
		return byte(code)
	}
}

func getServerIPMode(l logrus.FieldLogger) func(code ServerIPMode, options map[string]interface{}) byte {
	return func(codeString ServerIPMode, options map[string]interface{}) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options[serverIPCodeProperty]; !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", codeString, AddCharacterEntry)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", codeString, AddCharacterEntry)
			return 99
		}

		code, ok := codes[string(codeString)].(float64)
		if !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", codeString, AddCharacterEntry)
			return 99
		}
		return byte(code)
	}
}
