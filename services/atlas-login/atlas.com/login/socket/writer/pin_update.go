package writer

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"

	loginpkt "github.com/Chronicle20/atlas/libs/atlas-packet/login/clientbound"
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
