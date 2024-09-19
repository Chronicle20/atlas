package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
)

const PicResult = "PicResult"

func PicResultBody() BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(0)
		rtn := w.Bytes()
		//l.Debugf("Writing [%s] message. opcode [0x%02X].", PicResult, op&0xFF)
		return rtn
	}
}
