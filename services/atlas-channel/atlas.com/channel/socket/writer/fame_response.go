package writer

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	famepkt "github.com/Chronicle20/atlas-packet/fame"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const (
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
		return atlas_packet.ResolveCode(l, options, "operations", key)
	}
}
