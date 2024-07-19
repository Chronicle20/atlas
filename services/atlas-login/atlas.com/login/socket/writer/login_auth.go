package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const LoginAuth = "LoginAuth"

func LoginAuthBody(l logrus.FieldLogger) func(screen string) BodyProducer {
	return func(screen string) BodyProducer {
		return func(w *response.Writer, _ map[string]interface{}) []byte {
			w.WriteAsciiString(screen)
			rtn := w.Bytes()
			//l.Debugf("Writing [%s] message. opcode [0x%02X]. body={screen=%s}.", LoginAuth, op&0xFF, screen)
			return rtn
		}
	}
}
