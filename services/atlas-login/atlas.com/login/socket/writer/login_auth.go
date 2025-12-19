package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
)

const LoginAuth = "LoginAuth"

func LoginAuthBody(screen string) BodyProducer {
	return func(w *response.Writer, _ map[string]interface{}) []byte {
		w.WriteAsciiString(screen)
		rtn := w.Bytes()
		//l.Debugf("Writing [%s] message. opcode [0x%02X]. body={screen=%s}.", LoginAuth, op&0xFF, screen)
		return rtn
	}
}
