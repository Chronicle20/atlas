package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
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

func ServerLoadBody(code ServerLoadCode) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(ServerLoad, string(code), "codes", options))
			return w.Bytes()
		}
	}
}
