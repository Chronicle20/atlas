package writer

import (
	"strconv"

	"github.com/Chronicle20/atlas-socket/response"
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

func ReceiveFameResponseBody(l logrus.FieldLogger) func(fromName string, amount int8) BodyProducer {
	return func(fromName string, amount int8) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getFameOperation(l)(options, FameResponseReceive))
			w.WriteAsciiString(fromName)
			mode := (amount + 1) / 2
			w.WriteInt8(mode)
			return w.Bytes()
		}
	}
}

func GiveFameResponseBody(l logrus.FieldLogger) func(toName string, amount int8, total int16) BodyProducer {
	return func(toName string, amount int8, total int16) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getFameOperation(l)(options, FameResponseGive))
			w.WriteAsciiString(toName)
			mode := (amount + 1) / 2
			w.WriteInt8(mode)
			w.WriteInt16(total)
			w.WriteShort(0)
			return w.Bytes()
		}
	}
}

func FameResponseErrorBody(l logrus.FieldLogger) func(errCode string) BodyProducer {
	return func(errCode string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getFameOperation(l)(options, errCode))
			return w.Bytes()
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
