package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"

	loginpkt "github.com/Chronicle20/atlas-packet/login/clientbound"
)


type PinUpdateMode string

const (
	PinUpdateModeOk    PinUpdateMode = "OK"
	PinUpdateModeError PinUpdateMode = "ERROR"
)

func PinUpdateBody(mode PinUpdateMode) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			resolved := getCode(l)(loginpkt.PinUpdateWriter, string(mode), "modes", options)
			return loginpkt.NewPinUpdate(resolved).Encode(l, ctx)(options)
		}
	}
}
