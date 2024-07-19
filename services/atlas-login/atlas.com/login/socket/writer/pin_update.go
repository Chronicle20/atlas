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
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(PinUpdate, string(mode), "modes", options))
			rtn := w.Bytes()
			//l.Debugf("Writing [%s] message. opcode [0x%02X]. body={mode=%s}.", PinOperation, op&0xFF, mode)
			return rtn
		}
	}
}
