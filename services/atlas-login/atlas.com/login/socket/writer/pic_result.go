package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PicResult = "PicResult"

func PicResultBody(l logrus.FieldLogger) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(0)
		rtn := w.Bytes()
		//l.Debugf("Writing [%s] message. opcode [0x%02X].", PicResult, op&0xFF)
		return rtn
	}
}
