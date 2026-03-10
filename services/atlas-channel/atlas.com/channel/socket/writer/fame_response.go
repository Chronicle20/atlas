package writer

import (
	"context"
	"strconv"

	famepkt "github.com/Chronicle20/atlas-packet/fame"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const (
	FameResponse                         = "FameResponse"
	FameResponseReceive                  = "RECEIVE"
	FameResponseGive                     = "GIVE"
	FameResponseErrorTypeNotToday        = "NOT_TODAY"
	FameResponseErrorTypeNotThisMonth    = "NOT_THIS_MONTH"
	FameResponseErrorInvalidName         = "INVALID_NAME"
	FameResponseErrorTypeNotMinimumLevel = "NOT_MINIMUM_LEVEL"
	FameResponseErrorTypeUnexpected      = "UNEXPECTED"
)

func ReceiveFameResponseBody(fromName string, amount int8) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getFameOperation(l)(options, FameResponseReceive)
			return famepkt.NewReceiveFameResponse(mode, fromName, amount).Encode(l, ctx)(options)
		}
	}
}

func GiveFameResponseBody(toName string, amount int8, total int16) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getFameOperation(l)(options, FameResponseGive)
			return famepkt.NewGiveFameResponse(mode, toName, amount, total).Encode(l, ctx)(options)
		}
	}
}

func FameResponseErrorBody(errCode string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getFameOperation(l)(options, errCode)
			return famepkt.NewFameErrorResponse(mode).Encode(l, ctx)(options)
		}
	}
}

func getFameOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
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

		var code interface{}
		if code, ok = codes[key]; !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		op, err := strconv.ParseUint(code.(string), 0, 16)
		if err != nil {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}
		return byte(op)
	}
}
