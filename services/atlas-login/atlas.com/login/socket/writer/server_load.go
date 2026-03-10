package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"

	loginpkt "github.com/Chronicle20/atlas-packet/login"
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
		return func(options map[string]interface{}) []byte {
			resolved := getCode(l)(ServerLoad, string(code), "codes", options)
			return loginpkt.NewServerLoad(resolved).Encode(l, ctx)(options)
		}
	}
}
