package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ServerLoad = "ServerLoad"

type ServerLoadCode string

const (
	ServerLoadCodeOk             ServerLoadCode = "OK"
	ServerLoadCodeHighPopulation ServerLoadCode = "HIGH_POPULATION"
	ServerLoadCodeOverPopulated  ServerLoadCode = "OVER_POPULATED"
)

func ServerLoadBody(l logrus.FieldLogger) func(code ServerLoadCode) BodyProducer {
	return func(code ServerLoadCode) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(ServerLoad, string(code), "codes", options))
			return w.Bytes()
		}
	}
}
