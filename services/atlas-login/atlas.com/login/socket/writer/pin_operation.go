package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PinOperation = "PinOperation"

type PinOperationMode string

const (
	PinOperationModeOk               PinOperationMode = "OK"
	PinOperationModeRegister         PinOperationMode = "REGISTER"
	PinOperationModeInvalid          PinOperationMode = "INVALID"
	PinOperationModeConnectionFailed PinOperationMode = "CONNECTION_FAILED"
	PinOperationModeEnterEnterPin    PinOperationMode = "ENTER_PIN"
	PinOperationModeAlreadyLoggedIn  PinOperationMode = "ALREADY_LOGGED_IN"
)

func RegisterPinBody(l logrus.FieldLogger) BodyProducer {
	return PinOperationBody(l)(PinOperationModeRegister)
}

func RequestPinBody(l logrus.FieldLogger) BodyProducer {
	return PinOperationBody(l)(PinOperationModeEnterEnterPin)
}

func AcceptPinBody(l logrus.FieldLogger) BodyProducer {
	return PinOperationBody(l)(PinOperationModeOk)
}

func InvalidPinBody(l logrus.FieldLogger) BodyProducer {
	return PinOperationBody(l)(PinOperationModeInvalid)
}

func PinConnectionFailedBody(l logrus.FieldLogger) BodyProducer {
	return PinOperationBody(l)(PinOperationModeConnectionFailed)
}

func PinOperationBody(l logrus.FieldLogger) func(mode PinOperationMode) BodyProducer {
	return func(mode PinOperationMode) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(PinOperation, string(mode), "modes", options))
			rtn := w.Bytes()
			//l.Debugf("Writing [%s] message. opcode [0x%02X]. body={mode=%s}.", PinOperation, op&0xFF, mode)
			return rtn
		}
	}
}
