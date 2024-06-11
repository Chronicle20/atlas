package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const LoginAuth = "LoginAuth"

func LoginAuthBody(l logrus.FieldLogger) func(screen string) BodyProducer {
	return func(screen string) BodyProducer {
		return func(op uint16) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteAsciiString(screen)
			rtn := w.Bytes()
			l.Debugf("Writing [%s] message session. opcode [0x%d]. body={screen=%s}.", LoginAuth, op, screen)
			return rtn
		}
	}
}
