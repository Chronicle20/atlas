package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PinUpdate = "PinUpdate"

type PinUpdateMode string

const (
	PinUpdateModeOk    PinUpdateMode = "OK"
	PinUpdateModeError PinUpdateMode = "ERROR"
)

func PinUpdateBody(l logrus.FieldLogger) func(mode PinUpdateMode) BodyProducer {
	return func(mode PinUpdateMode) BodyProducer {
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteByte(getPinUpdateMode(l)(mode, options))
			rtn := w.Bytes()
			l.Debugf("Writing [%s] message. opcode [0x%02X]. body={mode=%s}.", PinOperation, op&0xFF, mode)
			return rtn
		}
	}
}

const pinPinUpdateProperty = "modes"

func getPinUpdateMode(l logrus.FieldLogger) func(code PinUpdateMode, options map[string]interface{}) byte {
	return func(codeString PinUpdateMode, options map[string]interface{}) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options[pinPinUpdateProperty]; !ok {
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
