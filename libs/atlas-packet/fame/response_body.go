package fame

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	"github.com/Chronicle20/atlas-packet/fame/clientbound"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

func ReceiveFameResponseBody(fromName string, amount int8) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "RECEIVE", func(mode byte) packet.Encoder {
		return clientbound.NewReceiveFameResponse(mode, fromName, amount)
	})
}

func GiveFameResponseBody(toName string, amount int8, total int16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", "GIVE", func(mode byte) packet.Encoder {
		return clientbound.NewGiveFameResponse(mode, toName, amount, total)
	})
}

func FameResponseErrorBody(errCode string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", errCode)
			return clientbound.NewFameErrorResponse(mode).Encode(l, ctx)(options)
		}
	}
}
