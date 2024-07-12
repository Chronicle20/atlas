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
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteByte(getPinOperationMode(l)(mode, options))
			rtn := w.Bytes()
			l.Debugf("Writing [%s] message. opcode [0x%02X]. body={mode=%s}.", PinOperation, op&0xFF, mode)
			return rtn
		}
	}
}

const pinOperationModeProperty = "modes"

func getPinOperationMode(l logrus.FieldLogger) func(code PinOperationMode, options map[string]interface{}) byte {
	return func(codeString PinOperationMode, options map[string]interface{}) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options[pinOperationModeProperty]; !ok {
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
