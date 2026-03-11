package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"

	loginpkt "github.com/Chronicle20/atlas-packet/login"
)


type PinOperationMode string

const (
	PinOperationModeOk               PinOperationMode = "OK"
	PinOperationModeRegister         PinOperationMode = "REGISTER"
	PinOperationModeInvalid          PinOperationMode = "INVALID"
	PinOperationModeConnectionFailed PinOperationMode = "CONNECTION_FAILED"
	PinOperationModeEnterEnterPin    PinOperationMode = "ENTER_PIN"
	PinOperationModeAlreadyLoggedIn  PinOperationMode = "ALREADY_LOGGED_IN"
)

func RegisterPinBody() packet.Encode {
	return PinOperationBody(PinOperationModeRegister)
}

func RequestPinBody() packet.Encode {
	return PinOperationBody(PinOperationModeEnterEnterPin)
}

func AcceptPinBody() packet.Encode {
	return PinOperationBody(PinOperationModeOk)
}

func InvalidPinBody() packet.Encode {
	return PinOperationBody(PinOperationModeInvalid)
}

func PinConnectionFailedBody() packet.Encode {
	return PinOperationBody(PinOperationModeConnectionFailed)
}

func PinOperationBody(mode PinOperationMode) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			resolved := getCode(l)(loginpkt.PinOperationWriter, string(mode), "modes", options)
			return loginpkt.NewPinOperation(resolved).Encode(l, ctx)(options)
		}
	}
}
